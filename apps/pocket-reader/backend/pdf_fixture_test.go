package readerd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPDFProviderExtractsGeneratedText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "generated.pdf")
	if err := os.WriteFile(path, []byte(minimalTextPDF("Generated Pocket Reader PDF phrase")), 0o600); err != nil {
		t.Fatal(err)
	}
	provider := PDFProvider{}
	doc, err := provider.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := provider.TOC(context.Background(), doc)
	if err != nil || len(pages) != 1 {
		t.Fatalf("pages=%v err=%v", pages, err)
	}
	hits, err := provider.Search(context.Background(), doc, "Pocket Reader")
	if err != nil || len(hits) == 0 {
		t.Fatalf("hits=%v err=%v", hits, err)
	}
	chunk, err := provider.Content(context.Background(), doc, Location{Loc: "0"})
	if err != nil || !strings.Contains(chunk.Content, "Generated") || chunk.HasNext || chunk.NextLoc != nil {
		t.Fatalf("chunk=%+v err=%v", chunk, err)
	}
}

func minimalTextPDF(text string) string {
	text = strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)").Replace(text)
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\nBT /F1 18 Tf 72 720 Td (%s) Tj ET\nendstream", len(fmt.Sprintf("BT /F1 18 Tf 72 720 Td (%s) Tj ET", text)), text),
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	positions := []int{0}
	for i, object := range objects {
		positions = append(positions, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(positions))
	for _, position := range positions[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", position)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(positions), xref)
	return b.String()
}
