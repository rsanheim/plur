package testruntime

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
)

// Store each field once per file instead of repeating its name for every example.
type exampleEntries []ExampleEntry
type exampleColumns struct {
	IDs       []string  `json:"id"`
	Lines     []int     `json:"line_number"`
	Locations []string  `json:"location_rerun_argument"`
	Runtimes  []float64 `json:"runtime_seconds"`
}

func (examples exampleEntries) MarshalJSONTo(enc *jsontext.Encoder) error {
	if len(examples) == 0 {
		return enc.WriteToken(jsontext.Null)
	}
	columns := exampleColumns{
		IDs:       make([]string, len(examples)),
		Lines:     make([]int, len(examples)),
		Locations: make([]string, len(examples)),
		Runtimes:  make([]float64, len(examples)),
	}
	for i, e := range examples {
		columns.IDs[i] = e.ID
		columns.Lines[i] = e.LineNumber
		columns.Locations[i] = e.LocationRerunArgument
		columns.Runtimes[i] = e.RuntimeSeconds
	}
	return json.MarshalEncode(enc, &columns)
}

func (examples *exampleEntries) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var columns *exampleColumns
	if err := json.UnmarshalDecode(dec, &columns); err != nil {
		return err
	}
	if columns == nil {
		*examples = nil
		return nil
	}
	n := len(columns.IDs)
	if len(columns.Lines) != n || len(columns.Locations) != n || len(columns.Runtimes) != n {
		return errors.New("mismatched example columns")
	}
	*examples = make(exampleEntries, n)
	for i := range *examples {
		(*examples)[i] = ExampleEntry{
			ID:                    columns.IDs[i],
			LineNumber:            columns.Lines[i],
			LocationRerunArgument: columns.Locations[i],
			RuntimeSeconds:        columns.Runtimes[i],
		}
	}
	return nil
}
