package gateway

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"
)

// multipartWriter wraps mime/multipart.Writer for buffered bodies.
type multipartWriter struct {
	*multipart.Writer
}

func newMultipartWriter(w io.Writer) *multipartWriter {
	return &multipartWriter{Writer: multipart.NewWriter(w)}
}

func (m *multipartWriter) writeFile(f FilePart) error {
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(f.Field), escapeQuotes(f.Filename)))
	if f.ContentType != "" {
		hdr.Set("Content-Type", f.ContentType)
	}
	part, err := m.CreatePart(hdr)
	if err != nil {
		return err
	}
	_, err = part.Write(f.Data)
	return err
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
