package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type pdfObj struct {
	dict        string
	stream      []byte
	skipEncrypt bool
}

var papersMM = map[string][2]float64{
	"A3": {297, 420}, "A4": {210, 297}, "A5": {148, 210},
	"B5": {176, 250}, "Letter": {215.9, 279.4}, "Legal": {215.9, 355.6},
}

const maxPagePt = 14400.0

func pageBox(p processedPage, o Options) (pageW, pageH, drawW, drawH, x, y float64) {
	switch o.PageMode {
	case "image":
		dx, dy := p.dpiX, p.dpiY
		if o.DpiSource == "fixed" && o.FixedDpi > 0 {
			dx, dy = o.FixedDpi, o.FixedDpi
		}
		if dx <= 0 {
			dx = o.DefaultDpi
		}
		if dy <= 0 {
			dy = o.DefaultDpi
		}
		if dx <= 0 {
			dx = 96
		}
		if dy <= 0 {
			dy = 96
		}
		pageW, pageH = float64(p.origW)/dx*72, float64(p.origH)/dy*72
		if o.MaxLongEdgeMm > 0 {
			limit := o.MaxLongEdgeMm * 72 / 25.4
			longest := pageW
			if pageH > longest {
				longest = pageH
			}
			if longest > limit && longest > 0 {
				k := limit / longest
				pageW *= k
				pageH *= k
			}
		}
	case "paper":
		size, ok := papersMM[o.Paper]
		if !ok {
			size = papersMM["A4"]
		}
		wmm, hmm := size[0], size[1]
		if (p.origW >= p.origH) != (wmm >= hmm) {
			wmm, hmm = hmm, wmm
		}
		pageW, pageH = wmm*72/25.4, hmm*72/25.4
	default:
		pageW, pageH = float64(p.w)*0.75, float64(p.h)*0.75
	}
	if pageW <= 0 {
		pageW = 1
	}
	if pageH <= 0 {
		pageH = 1
	}
	if pageW > maxPagePt || pageH > maxPagePt {
		k := maxPagePt / pageW
		if pageH > pageW {
			k = maxPagePt / pageH
		}
		pageW *= k
		pageH *= k
	}

	margin := 0.0
	if o.PageMode != "fit" {
		margin = o.MarginMm * 72 / 25.4
		if margin <= 0 && o.PageMode == "paper" {
			margin = 8
		}
	}
	availW, availH := pageW-2*margin, pageH-2*margin
	if availW <= 0 || availH <= 0 {
		availW, availH = pageW, pageH
		margin = 0
	}
	s := availW / float64(p.w)
	if s*float64(p.h) > availH {
		s = availH / float64(p.h)
	}
	drawW, drawH = float64(p.w)*s, float64(p.h)*s
	x, y = margin+(availW-drawW)/2, margin+(availH-drawH)/2
	return
}

func pdfString(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)", "\r", " ", "\n", " ")
	return "(" + r.Replace(s) + ")"
}

func buildPDF(pages []processedPage, o Options) []pdfObj {
	n := len(pages)
	kids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", 3+i*3))
	}
	objs := make([]pdfObj, 0, 2+3*n+2+n)
	objs = append(objs, pdfObj{dict: "<< /Type /Catalog /Pages 2 0 R >>\n"})
	objs = append(objs, pdfObj{
		dict: fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>\n", strings.Join(kids, " "), n),
	})

	for i, p := range pages {
		pageNum := 3 + i*3
		contentNum, imageNum := pageNum+1, pageNum+2
		pageW, pageH, drawW, drawH, x, y := pageBox(p, o)
		pageDict := fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] "+
				"/Resources << /XObject << /Im0 %d 0 R >> >> /Contents %d 0 R >>\n",
			pageW, pageH, imageNum, contentNum)
		objs = append(objs, pdfObj{dict: pageDict})
		content := fmt.Sprintf("q\n%.4f 0 0 %.4f %.4f %.4f cm\n/Im0 Do\nQ\n", drawW, drawH, x, y)
		objs = append(objs, pdfObj{dict: "<< /Length %LEN% >>\n", stream: []byte(content)})
		imageDict := fmt.Sprintf(
			"<< /Type /XObject /Subtype /Image /Width %d /Height %d "+
				"/ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %%LEN%% >>\n",
			p.w, p.h)
		objs = append(objs, pdfObj{dict: imageDict, stream: p.jpeg})
	}

	catalogExtra := ""
	if o.Bookmarks && n > 0 {
		rootNum := len(objs) + 1
		first, last := rootNum+1, rootNum+n
		objs = append(objs, pdfObj{dict: fmt.Sprintf(
			"<< /Type /Outlines /First %d 0 R /Last %d 0 R /Count %d >>\n", first, last, n)})
		for i, p := range pages {
			itemNum := rootNum + 1 + i
			prev, next := "", ""
			if i > 0 {
				prev = fmt.Sprintf(" /Prev %d 0 R", itemNum-1)
			}
			if i < n-1 {
				next = fmt.Sprintf(" /Next %d 0 R", itemNum+1)
			}
			title := p.name
			if title == "" {
				title = fmt.Sprintf("Page %d", i+1)
			}
			objs = append(objs, pdfObj{dict: fmt.Sprintf(
				"<< /Title %s /Parent %d 0 R%s%s /Dest [%d 0 R /XYZ 0 0 0] >>\n",
				pdfString(title), rootNum, prev, next, 3+i*3)})
		}
		catalogExtra += fmt.Sprintf(" /Outlines %d 0 R", rootNum)
	}

	infoNum := len(objs) + 1
	title := o.Title
	if title == "" {
		title = "图片合集"
	}
	objs = append(objs, pdfObj{dict: fmt.Sprintf(
		"<< /Title %s /Producer (%s) /Creator (%s) /CreationDate (D:%s) >>\n",
		pdfString(title), ProductName, ProductName, time.Now().Format("20060102150405"))})
	catalogExtra += fmt.Sprintf(" /Info %d 0 R", infoNum)
	objs[0].dict = fmt.Sprintf("<< /Type /Catalog /Pages 2 0 R%s >>\n", catalogExtra)
	return objs
}

func renderPDF(objs []pdfObj, enc *pdfEncryption, encryptObjNum int) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, 0, len(objs)+1)
	offsets = append(offsets, 0)
	for i, o := range objs {
		offsets = append(offsets, buf.Len())
		data := o.stream
		if enc != nil && !o.skipEncrypt && data != nil {
			data = enc.encryptStream(data)
		}
		dict := o.dict
		if strings.Contains(dict, "%LEN%") {
			dict = strings.ReplaceAll(dict, "%LEN%", fmt.Sprintf("%d", len(data)))
		}
		fmt.Fprintf(&buf, "%d 0 obj\n%s", i+1, dict)
		if data != nil {
			buf.WriteString("stream\n")
			buf.Write(data)
			buf.WriteString("\nendstream\n")
		}
		buf.WriteString("endobj\n")
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objs)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets[1:] {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	id1, id2 := make([]byte, 16), make([]byte, 16)
	if _, err := rand.Read(id1); err != nil {
		return nil, err
	}
	if _, err := rand.Read(id2); err != nil {
		return nil, err
	}
	encRef := ""
	if encryptObjNum > 0 {
		encRef = fmt.Sprintf(" /Encrypt %d 0 R", encryptObjNum)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R /ID [<%s><%s>]%s >>\nstartxref\n%d\n%%%%EOF\n",
		len(objs)+1, hex.EncodeToString(id1), hex.EncodeToString(id2), encRef, xref)
	return buf.Bytes(), nil
}
