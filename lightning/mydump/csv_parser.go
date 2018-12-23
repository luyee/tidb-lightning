package mydump

import (
	"bytes"
	"io"

	"github.com/pingcap/tidb-lightning/lightning/config"
	"github.com/pingcap/tidb/util/hack"
	"github.com/pkg/errors"
)

type CSVParser struct {
	blockParser

	rowBuf bytes.Buffer
	cfg    *config.CSVConfig

	delim1 []byte
	delim2 []byte
}

func NewCSVParser(cfg *config.CSVConfig, reader io.Reader) *CSVParser {
	return &CSVParser{
		blockParser: makeBlockParser(reader),
		cfg:         cfg,

		delim1: []byte(cfg.Delimiter),
		delim2: []byte(cfg.Delimiter + cfg.Delimiter),
	}
}

type csvToken byte

const (
	csvTokNil csvToken = iota
	csvTokSep
	csvTokNewLine
	csvTokField
	csvTokQuotedField
)

func (parser *CSVParser) appendEmptyValues(sepCount int) {
	var content string
	if !parser.cfg.NotNull && parser.cfg.Null == "" {
		content = ",NULL"
	} else {
		content = ",''"
	}
	for i := 1; i < sepCount; i++ {
		parser.rowBuf.WriteString(content)
	}
}

func (parser *CSVParser) appendField(content []byte, shouldUnquote bool, quote byte) {
	if !parser.cfg.BackslashEscape {
		content = bytes.Replace(content, []byte(`\`), []byte(`\\`), -1)
	}
	if shouldUnquote {
		content = bytes.Replace(content[1:len(content)-1], parser.delim2, parser.delim1, -1)
	}
	if !parser.cfg.NotNull && parser.cfg.Null == hack.String(content) {
		parser.rowBuf.WriteString(",NULL")
		return
	}

	parser.rowBuf.WriteByte(',')
	parser.rowBuf.WriteByte(quote)
	parser.rowBuf.Write(bytes.Replace(content, []byte{quote}, []byte{quote, quote}, -1))
	parser.rowBuf.WriteByte(quote)
}

// ReadRow reads a row from the datafile.
func (parser *CSVParser) ReadRow() error {
	emptySepCount := 0
	hasField := false

	// skip the header first
	if parser.pos == 0 && parser.cfg.Header {
	outside:
		for {
			tok, content, err := parser.lex()
			if err != nil {
				return errors.Trace(err)
			}
			switch tok {
			case csvTokSep:
			case csvTokField:
				parser.appendField(content, false, '`')
			case csvTokQuotedField:
				parser.appendField(content, true, '`')
			case csvTokNewLine:
				len := parser.rowBuf.Len()
				parser.columns = make([]byte, len+1)
				copy(parser.columns[:len], parser.rowBuf.Bytes())
				parser.columns[0] = '('
				parser.columns[len] = ')'
				parser.rowBuf.Reset()
				break outside
			}
		}
	}

	for {
		tok, content, err := parser.lex()
		switch errors.Cause(err) {
		case nil:
			break
		case io.EOF:
			if hasField {
				tok = csvTokNewLine
				break
			}
			fallthrough
		default:
			return errors.Trace(err)
		}

		hasField = true

		switch tok {
		case csvTokSep:
			emptySepCount++

		case csvTokField:
			parser.appendEmptyValues(emptySepCount)
			emptySepCount = 0
			parser.appendField(content, false, '\'')

		case csvTokQuotedField:
			parser.appendEmptyValues(emptySepCount)
			emptySepCount = 0
			parser.appendField(content, true, '\'')

		case csvTokNewLine:
			if !parser.cfg.TrimLastSep {
				parser.appendEmptyValues(emptySepCount)
			}
			parser.rowBuf.WriteByte(')')
			parser.lastRow.Row = parser.rowBuf.Bytes()
			parser.lastRow.Row[0] = '('
			parser.lastRow.RowID++
			parser.rowBuf.Reset()
			return nil
		}
	}
}