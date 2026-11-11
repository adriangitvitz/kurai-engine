package ranker

import (
	"testing"
)

func TestEntityRecognizer_LoadDefaults(t *testing.T) {
	er := NewEntityRecognizer()
	er.LoadDefaults()

	stats := er.Stats()
	if stats["products"] == 0 {
		t.Error("No products loaded")
	}
	if stats["organizations"] == 0 {
		t.Error("No organizations loaded")
	}
	if stats["languages"] == 0 {
		t.Error("No languages loaded")
	}
	if stats["frameworks"] == 0 {
		t.Error("No frameworks loaded")
	}
}

func TestEntityRecognizer_Recognize(t *testing.T) {
	er := NewEntityRecognizer()
	er.LoadDefaults()

	tests := []struct {
		name      string
		terms     []string
		wantType  string
		wantCanon string
	}{
		{"kubernetes variant", []string{"k8s"}, "product", "Kubernetes"},
		{"go language", []string{"golang"}, "language", "Go"},
		{"python language", []string{"python"}, "language", "Python"},
		{"react framework", []string{"react"}, "framework", "React"},
		{"github org", []string{"github"}, "organization", "GitHub"},
		{"docker product", []string{"docker"}, "product", "Docker"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := er.Recognize(tt.terms)
			if len(entities) == 0 {
				t.Errorf("Recognize(%v) returned no entities", tt.terms)
				return
			}

			found := false
			for _, e := range entities {
				if e.Type == tt.wantType && e.Canonical == tt.wantCanon {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Recognize(%v) did not find type=%s, canonical=%s", tt.terms, tt.wantType, tt.wantCanon)
			}
		})
	}
}

func TestEntityRecognizer_GetCanonical(t *testing.T) {
	er := NewEntityRecognizer()
	er.LoadDefaults()

	tests := []struct {
		term      string
		wantCanon string
		wantType  EntityType
	}{
		{"k8s", "Kubernetes", EntityTypeProduct},
		{"golang", "Go", EntityTypeLanguage},
		{"react", "React", EntityTypeFramework},
		{"github", "GitHub", EntityTypeOrganization},
		{"unknown", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.term, func(t *testing.T) {
			canon, typ := er.GetCanonical(tt.term)
			if canon != tt.wantCanon || typ != tt.wantType {
				t.Errorf("GetCanonical(%q) = (%q, %q), want (%q, %q)",
					tt.term, canon, typ, tt.wantCanon, tt.wantType)
			}
		})
	}
}

func TestEntityRecognizer_AddEntities(t *testing.T) {
	er := NewEntityRecognizer()

	er.AddProduct("CustomDB", []string{"customdb", "cdb"})
	er.AddLanguage("CustomLang", []string{"customlang", "cl"})
	er.AddFramework("CustomFramework", []string{"customfw"})
	er.AddOrganization("CustomOrg", []string{"customorg"})

	tests := []struct {
		term      string
		wantCanon string
		wantType  EntityType
	}{
		{"cdb", "CustomDB", EntityTypeProduct},
		{"customlang", "CustomLang", EntityTypeLanguage},
		{"customfw", "CustomFramework", EntityTypeFramework},
		{"customorg", "CustomOrg", EntityTypeOrganization},
	}

	for _, tt := range tests {
		t.Run(tt.term, func(t *testing.T) {
			canon, typ := er.GetCanonical(tt.term)
			if canon != tt.wantCanon || typ != tt.wantType {
				t.Errorf("GetCanonical(%q) = (%q, %q), want (%q, %q)",
					tt.term, canon, typ, tt.wantCanon, tt.wantType)
			}
		})
	}
}

func TestEntityRecognizer_LoadFromYAML(t *testing.T) {
	yamlData := []byte(`
products:
  - name: TestProduct
    variants: [testp, tp]
languages:
  - name: TestLang
    variants: [testl]
frameworks:
  - name: TestFramework
    variants: [testf]
organizations:
  - name: TestOrg
    variants: [testo]
`)

	er := NewEntityRecognizer()
	err := er.LoadFromYAML(yamlData)
	if err != nil {
		t.Fatalf("LoadFromYAML() error = %v", err)
	}

	tests := []struct {
		term      string
		wantCanon string
		wantType  EntityType
	}{
		{"testp", "TestProduct", EntityTypeProduct},
		{"testl", "TestLang", EntityTypeLanguage},
		{"testf", "TestFramework", EntityTypeFramework},
		{"testo", "TestOrg", EntityTypeOrganization},
	}

	for _, tt := range tests {
		t.Run(tt.term, func(t *testing.T) {
			canon, typ := er.GetCanonical(tt.term)
			if canon != tt.wantCanon || typ != tt.wantType {
				t.Errorf("GetCanonical(%q) = (%q, %q), want (%q, %q)",
					tt.term, canon, typ, tt.wantCanon, tt.wantType)
			}
		})
	}
}

func TestEntityRecognizer_RecognizeWithContext(t *testing.T) {
	er := NewEntityRecognizer()
	er.AddFramework("React Native", []string{"react native", "reactnative"})

	entities := er.RecognizeWithContext("how to use react native")

	found := false
	for _, e := range entities {
		if e.Canonical == "React Native" {
			found = true
			break
		}
	}

	if !found {
		t.Error("RecognizeWithContext() did not find 'React Native' multi-word entity")
	}
}

func TestEntityRecognizer_CaseInsensitive(t *testing.T) {
	er := NewEntityRecognizer()
	er.LoadDefaults()

	cases := []string{"python", "Python", "PYTHON", "PyThOn"}
	for _, c := range cases {
		canon, typ := er.GetCanonical(c)
		if canon != "Python" || typ != EntityTypeLanguage {
			t.Errorf("GetCanonical(%q) = (%q, %q), want (Python, language)", c, canon, typ)
		}
	}
}
