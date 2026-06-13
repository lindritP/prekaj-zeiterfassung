// Package pdf renders PDFs (Monatsbericht) with maroto v2 — pure Go, no external binaries.
package pdf

import (
	"fmt"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// BerichtZeile is one time-booking line in the report.
type BerichtZeile struct {
	Datum     string
	Baustelle string
	Start     string
	Ende      string
	PauseMin  int32
	Dauer     string
}

// MonatsberichtData is the input for a monthly report PDF.
type MonatsberichtData struct {
	ArbeiterName    string
	Jahr            int
	Monat           int
	Zeilen          []BerichtZeile
	IstMinuten      int64
	SollMinuten     int64
	SaldoMinuten    int64
	UrlaubKrankTage int
}

// Monatsbericht renders the monthly report and returns the PDF bytes.
func Monatsbericht(d MonatsberichtData) ([]byte, error) {
	cfg := config.NewBuilder().WithPageSize(pagesize.A4).Build()
	m := maroto.New(cfg)

	m.AddRows(
		text.NewRow(12, "FliesenPrekaj GmbH — Monatsbericht",
			props.Text{Size: 16, Style: fontstyle.Bold, Align: align.Center}),
		text.NewRow(7, fmt.Sprintf("%s   ·   %02d/%d", d.ArbeiterName, d.Monat, d.Jahr),
			props.Text{Size: 11, Align: align.Center}),
	)

	header := props.Text{Style: fontstyle.Bold, Size: 9}
	m.AddRow(7,
		col.New(2).Add(text.New("Datum", header)),
		col.New(3).Add(text.New("Baustelle", header)),
		col.New(2).Add(text.New("Start", header)),
		col.New(2).Add(text.New("Ende", header)),
		col.New(1).Add(text.New("Pause", header)),
		col.New(2).Add(text.New("Dauer", header)),
	)

	cell := props.Text{Size: 9}
	if len(d.Zeilen) == 0 {
		m.AddRows(text.NewRow(6, "Keine Zeitbuchungen in diesem Monat.",
			props.Text{Size: 9, Align: align.Center}))
	}
	for _, z := range d.Zeilen {
		m.AddRow(6,
			col.New(2).Add(text.New(z.Datum, cell)),
			col.New(3).Add(text.New(z.Baustelle, cell)),
			col.New(2).Add(text.New(z.Start, cell)),
			col.New(2).Add(text.New(z.Ende, cell)),
			col.New(1).Add(text.New(fmt.Sprintf("%d", z.PauseMin), cell)),
			col.New(2).Add(text.New(z.Dauer, cell)),
		)
	}

	m.AddRows(
		text.NewRow(10, fmt.Sprintf("Ist: %s    Soll: %s    Saldo: %s    Urlaub/Krank: %d Tage",
			hhmm(d.IstMinuten), hhmm(d.SollMinuten), hhmmSigned(d.SaldoMinuten), d.UrlaubKrankTage),
			props.Text{Size: 11, Style: fontstyle.Bold, Top: 6}),
	)

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return doc.GetBytes(), nil
}

// hhmm formats minutes as "H:MM h" (absolute value).
func hhmm(min int64) string {
	if min < 0 {
		min = -min
	}
	return fmt.Sprintf("%d:%02d h", min/60, min%60)
}

// hhmmSigned formats minutes as "H:MM h" with a leading minus for negatives.
func hhmmSigned(min int64) string {
	if min < 0 {
		return "-" + hhmm(min)
	}
	return hhmm(min)
}
