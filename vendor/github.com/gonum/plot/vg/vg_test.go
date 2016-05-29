// Copyright ©2015 The gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vg_test

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"rsc.io/pdf"
)

var generateTestData = flag.Bool("regen", false, "Uses the current state to regenerate the test data.")

// TestLineWidth tests output against test images generated by
// running tests with -tag good.
func TestLineWidth(t *testing.T) {
	formats := []string{
		// TODO: Add logic to cope with run to run eps differences.
		"pdf",
		"svg",
		"png",
		"tiff",
		"jpg",
	}

	const (
		width  = 100
		height = 100
	)

	for _, w := range []vg.Length{-1, 0, 1} {
		for _, typ := range formats {
			p, err := lines(w)
			if err != nil {
				log.Fatalf("failed to create plot for %v:%s: %v", w, typ, err)
			}

			c, err := p.WriterTo(width, height, typ)
			if err != nil {
				t.Fatalf("failed to render plot for %v:%s: %v", w, typ, err)
			}

			var buf bytes.Buffer
			if _, err = c.WriteTo(&buf); err != nil {
				t.Fatalf("failed to write plot for %v:%s: %v", w, typ, err)
			}

			name := filepath.Join(".", "testdata", fmt.Sprintf("width_%v.%s", w, typ))

			// Recreate Golden images.
			if *generateTestData {
				err = p.Save(width, height, name)
				if err != nil {
					log.Fatalf("failed to save %q: %v", name, err)
				}
			}

			switch typ {
			case "svg":
				want, err := ioutil.ReadFile(name)
				if err != nil {
					t.Fatalf("failed to read test image: %v", err)
				}

				if !bytes.Equal(buf.Bytes(), want) {
					t.Errorf("image mismatch for %v:%s", w, typ)
				}

			case "pdf":
				f, err := os.Open(name)
				if err != nil {
					t.Fatalf("failed to open test image: %v", err)
				}
				defer f.Close()
				fi, err := f.Stat()
				if err != nil {
					t.Fatalf("failed to retrieve test image infos: %v", err)
				}
				want, err := pdf.NewReader(f, fi.Size())
				if err != nil {
					t.Fatalf("failed to decode test image (typ=%s): %v", typ, err)
				}

				r := bytes.NewReader(buf.Bytes())
				// TODO(sbinet): bytes.Reader.Size was introduced only after go-1.4
				// use that if/when we drop go-1.4 bwd compat.
				sz := int64(len(buf.Bytes()))
				got, err := pdf.NewReader(r, sz)
				if err != nil {
					t.Fatalf("failed to decode image (typ=%s): %v", typ, err)
				}

				if !cmpPdf(got, want) {
					t.Errorf("image mismatch for %v:%s", w, typ)
				}

			default:
				f, err := os.Open(name)
				if err != nil {
					t.Fatalf("failed to open test image: %v", err)
				}
				defer f.Close()

				want, _, err := image.Decode(f)
				if err != nil {
					t.Fatalf("failed to read test image (typ=%s): %v", typ, err)
				}

				got, _, err := image.Decode(&buf)
				if err != nil {
					t.Fatalf("failed to decode image (typ=%s): %v", typ, err)
				}

				if !reflect.DeepEqual(got, want) {
					t.Errorf("image mismatch for %v:%s", w, typ)
				}
			}
		}
	}
}

func cmpPdf(pdf1, pdf2 *pdf.Reader) bool {
	n1 := pdf1.NumPage()
	n2 := pdf2.NumPage()
	if n1 != n2 {
		return false
	}

	for i := 1; i <= n1; i++ {
		p1 := pdf1.Page(i).Content()
		p2 := pdf2.Page(i).Content()
		if !reflect.DeepEqual(p1, p2) {
			return false
		}
	}

	t1 := pdf1.Trailer().String()
	t2 := pdf2.Trailer().String()
	return t1 == t2
}

func lines(w vg.Length) (*plot.Plot, error) {
	p, err := plot.New()
	if err != nil {
		return nil, err
	}

	pts := plotter.XYs{{0, 0}, {0, 1}, {1, 0}, {1, 1}}
	line, err := plotter.NewLine(pts)
	line.Width = w
	if err != nil {
		return nil, err
	}
	p.Add(line)

	return p, nil
}

func TestParseLength(t *testing.T) {
	for _, table := range []struct {
		str  string
		want vg.Length
		err  error
	}{
		{
			str:  "42.2cm",
			want: 42.2 * vg.Centimeter,
		},
		{
			str:  "42.2mm",
			want: 42.2 * vg.Millimeter,
		},
		{
			str:  "42.2in",
			want: 42.2 * vg.Inch,
		},
		{
			str:  "42.2pt",
			want: 42.2,
		},
		{
			str:  "42.2",
			want: 42.2,
		},
		{
			str: "999bottles",
			err: fmt.Errorf(`strconv.ParseFloat: parsing "999bottles": invalid syntax`),
		},
		{
			str:  "42inch",
			want: 42 * vg.Inch,
			err:  fmt.Errorf(`strconv.ParseFloat: parsing "42inch": invalid syntax`),
		},
	} {
		v, err := vg.ParseLength(table.str)
		if table.err != nil {
			if err == nil {
				t.Errorf("%s: expected an error (%v)\n",
					table.str, table.err,
				)
			}
			if table.err.Error() != err.Error() {
				t.Errorf("%s: got error=%q. want=%q\n",
					table.str, err.Error(), table.err.Error(),
				)
			}
			continue
		}
		if err != nil {
			t.Errorf("error setting flag.Value %q: %v\n",
				table.str,
				err,
			)
		}
		if v != table.want {
			t.Errorf("%s: incorrect value. got %v, want %v\n",
				table.str,
				float64(v), float64(table.want),
			)
		}
	}
}