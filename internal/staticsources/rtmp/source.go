// Package rtmp contains the RTMP static source.
package rtmp

import (
	"context"
	ctls "crypto/tls"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"

	"github.com/most-1995/mediamtx/internal/conf"
	"github.com/most-1995/mediamtx/internal/defs"
	"github.com/most-1995/mediamtx/internal/logger"
	"github.com/most-1995/mediamtx/internal/protocols/rtmp"
	"github.com/most-1995/mediamtx/internal/protocols/tls"
	"github.com/most-1995/mediamtx/internal/stream"
	"github.com/most-1995/mediamtx/internal/unit"
)

// Source is a RTMP static source.
type Source struct {
	ResolvedSource string
	ReadTimeout    conf.StringDuration
	WriteTimeout   conf.StringDuration
	Parent         defs.StaticSourceParent
}

// Log implements logger.Writer.
func (s *Source) Log(level logger.Level, format string, args ...interface{}) {
	s.Parent.Log(level, "[RTMP source] "+format, args...)
}

// Run implements StaticSource.
func (s *Source) Run(params defs.StaticSourceRunParams) error {
	s.Log(logger.Debug, "connecting")

	u, err := url.Parse(s.ResolvedSource)
	if err != nil {
		return err
	}

	// add default port
	_, _, err = net.SplitHostPort(u.Host)
	if err != nil {
		if u.Scheme == "rtmp" {
			u.Host = net.JoinHostPort(u.Host, "1935")
		} else {
			u.Host = net.JoinHostPort(u.Host, "1936")
		}
	}

	nconn, err := func() (net.Conn, error) {
		ctx2, cancel2 := context.WithTimeout(params.Context, time.Duration(s.ReadTimeout))
		defer cancel2()

		if u.Scheme == "rtmp" {
			return (&net.Dialer{}).DialContext(ctx2, "tcp", u.Host)
		}

		return (&ctls.Dialer{
			Config: tls.ConfigForFingerprint(params.Conf.SourceFingerprint),
		}).DialContext(ctx2, "tcp", u.Host)
	}()
	if err != nil {
		return err
	}

	readDone := make(chan error)
	go func() {
		readDone <- s.runReader(u, nconn)
	}()

	for {
		select {
		case err := <-readDone:
			nconn.Close()
			return err

		case <-params.ReloadConf:

		case <-params.Context.Done():
			nconn.Close()
			<-readDone
			return nil
		}
	}
}

func (s *Source) runReader(u *url.URL, nconn net.Conn) error {
	nconn.SetReadDeadline(time.Now().Add(time.Duration(s.ReadTimeout)))
	nconn.SetWriteDeadline(time.Now().Add(time.Duration(s.WriteTimeout)))
	conn, err := rtmp.NewClientConn(nconn, u, false)
	if err != nil {
		return err
	}

	mc, err := rtmp.NewReader(conn)
	if err != nil {
		return err
	}

	videoFormat, audioFormat := mc.Tracks()

	var medias []*description.Media
	var stream *stream.Stream

	if videoFormat != nil {
		videoMedia := &description.Media{
			Type:    description.MediaTypeVideo,
			Formats: []format.Format{videoFormat},
		}
		medias = append(medias, videoMedia)

		switch videoFormat.(type) {
		case *format.H264:
			mc.OnDataH264(func(pts time.Duration, au [][]byte) {
				stream.WriteUnit(videoMedia, videoFormat, &unit.H264{
					Base: unit.Base{
						NTP: time.Now(),
						PTS: pts,
					},
					AU: au,
				})
			})

		default:
			return fmt.Errorf("unsupported video codec: %T", videoFormat)
		}
	}

	if audioFormat != nil { //nolint:dupl
		audioMedia := &description.Media{
			Type:    description.MediaTypeAudio,
			Formats: []format.Format{audioFormat},
		}
		medias = append(medias, audioMedia)

		switch audioFormat.(type) {
		case *format.MPEG4Audio:
			mc.OnDataMPEG4Audio(func(pts time.Duration, au []byte) {
				stream.WriteUnit(audioMedia, audioFormat, &unit.MPEG4Audio{
					Base: unit.Base{
						NTP: time.Now(),
						PTS: pts,
					},
					AUs: [][]byte{au},
				})
			})

		case *format.MPEG1Audio:
			mc.OnDataMPEG1Audio(func(pts time.Duration, frame []byte) {
				stream.WriteUnit(audioMedia, audioFormat, &unit.MPEG1Audio{
					Base: unit.Base{
						NTP: time.Now(),
						PTS: pts,
					},
					Frames: [][]byte{frame},
				})
			})

		default:
			return fmt.Errorf("unsupported audio codec: %T", audioFormat)
		}
	}

	res := s.Parent.SetReady(defs.PathSourceStaticSetReadyReq{
		Desc:               &description.Session{Medias: medias},
		GenerateRTPPackets: true,
	})
	if res.Err != nil {
		return res.Err
	}

	defer s.Parent.SetNotReady(defs.PathSourceStaticSetNotReadyReq{})

	stream = res.Stream

	// disable write deadline to allow outgoing acknowledges
	nconn.SetWriteDeadline(time.Time{})

	for {
		nconn.SetReadDeadline(time.Now().Add(time.Duration(s.ReadTimeout)))
		err := mc.Read()
		if err != nil {
			return err
		}
	}
}

// APISourceDescribe implements StaticSource.
func (*Source) APISourceDescribe() defs.APIPathSourceOrReader {
	return defs.APIPathSourceOrReader{
		Type: "rtmpSource",
		ID:   "",
	}
}
