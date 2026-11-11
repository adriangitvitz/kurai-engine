package ranker

import (
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type EntityType string

const (
	EntityTypeProduct      EntityType = "product"
	EntityTypeOrganization EntityType = "organization"
	EntityTypeLanguage     EntityType = "language"
	EntityTypeFramework    EntityType = "framework"
)

type EntityDefinition struct {
	Name     string   `yaml:"name"`
	Variants []string `yaml:"variants"`
}

type EntityConfig struct {
	Products      []EntityDefinition `yaml:"products"`
	Organizations []EntityDefinition `yaml:"organizations"`
	Languages     []EntityDefinition `yaml:"languages"`
	Frameworks    []EntityDefinition `yaml:"frameworks"`
}

type EntityRecognizer struct {
	mu sync.RWMutex

	products      map[string]string
	organizations map[string]string
	languages     map[string]string
	frameworks    map[string]string
}

func NewEntityRecognizer() *EntityRecognizer {
	return &EntityRecognizer{
		products:      make(map[string]string),
		organizations: make(map[string]string),
		languages:     make(map[string]string),
		frameworks:    make(map[string]string),
	}
}

func (er *EntityRecognizer) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return er.LoadFromYAML(data)
}

func (er *EntityRecognizer) LoadFromYAML(data []byte) error {
	var config EntityConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	er.mu.Lock()
	defer er.mu.Unlock()

	for _, entity := range config.Products {
		er.addEntity(er.products, entity)
	}

	for _, entity := range config.Organizations {
		er.addEntity(er.organizations, entity)
	}

	for _, entity := range config.Languages {
		er.addEntity(er.languages, entity)
	}

	for _, entity := range config.Frameworks {
		er.addEntity(er.frameworks, entity)
	}

	return nil
}

func (er *EntityRecognizer) addEntity(m map[string]string, entity EntityDefinition) {
	m[strings.ToLower(entity.Name)] = entity.Name

	for _, variant := range entity.Variants {
		m[strings.ToLower(variant)] = entity.Name
	}
}

func (er *EntityRecognizer) LoadDefaults() {
	er.mu.Lock()
	defer er.mu.Unlock()

	defaultProducts := map[string]string{
		"postgres":       "PostgreSQL",
		"postgresql":     "PostgreSQL",
		"pg":             "PostgreSQL",
		"mysql":          "MySQL",
		"mariadb":        "MySQL",
		"mongodb":        "MongoDB",
		"mongo":          "MongoDB",
		"redis":          "Redis",
		"elasticsearch":  "Elasticsearch",
		"elastic":        "Elasticsearch",
		"es":             "Elasticsearch",
		"sqlite":         "SQLite",
		"sqlite3":        "SQLite",
		"docker":         "Docker",
		"dockerfile":     "Docker",
		"docker-compose": "Docker",
		"kubernetes":     "Kubernetes",
		"k8s":            "Kubernetes",
		"kubectl":        "Kubernetes",
		"kube":           "Kubernetes",
		"terraform":      "Terraform",
		"tf":             "Terraform",
		"aws":            "AWS",
		"gcp":            "Google Cloud",
		"azure":          "Azure",
		"nginx":          "Nginx",
		"apache":         "Apache",
		"kafka":          "Kafka",
		"rabbitmq":       "RabbitMQ",
		"vscode":         "VS Code",
		"vim":            "Vim",
		"neovim":         "Vim",
		"nvim":           "Vim",
		"openai":         "OpenAI",
		"chatgpt":        "OpenAI",
		"gpt":            "OpenAI",
		"gpt-4":          "OpenAI",
		"claude":         "Anthropic",
		"anthropic":      "Anthropic",
		"tensorflow":     "TensorFlow",
		"pytorch":        "PyTorch",
		"torch":          "PyTorch",
	}
	for k, v := range defaultProducts {
		er.products[k] = v
	}

	defaultOrgs := map[string]string{
		"google":     "Google",
		"alphabet":   "Google",
		"microsoft":  "Microsoft",
		"msft":       "Microsoft",
		"apple":      "Apple",
		"meta":       "Meta",
		"facebook":   "Meta",
		"fb":         "Meta",
		"amazon":     "Amazon",
		"amzn":       "Amazon",
		"netflix":    "Netflix",
		"github":     "GitHub",
		"gh":         "GitHub",
		"gitlab":     "GitLab",
		"cloudflare": "Cloudflare",
		"cf":         "Cloudflare",
		"nvidia":     "NVIDIA",
		"amd":        "AMD",
		"intel":      "Intel",
		"twitter":    "Twitter/X",
		"x":          "Twitter/X",
		"linkedin":   "LinkedIn",
		"stripe":     "Stripe",
		"hashicorp":  "HashiCorp",
		"redhat":     "Red Hat",
		"canonical":  "Canonical",
		"ubuntu":     "Canonical",
	}
	for k, v := range defaultOrgs {
		er.organizations[k] = v
	}

	defaultLangs := map[string]string{
		"go":         "Go",
		"golang":     "Go",
		"python":     "Python",
		"py":         "Python",
		"python3":    "Python",
		"javascript": "JavaScript",
		"js":         "JavaScript",
		"ecmascript": "JavaScript",
		"typescript": "TypeScript",
		"ts":         "TypeScript",
		"rust":       "Rust",
		"rustlang":   "Rust",
		"java":       "Java",
		"jdk":        "Java",
		"jvm":        "Java",
		"kotlin":     "Kotlin",
		"swift":      "Swift",
		"c":          "C",
		"c++":        "C++",
		"cpp":        "C++",
		"cxx":        "C++",
		"c#":         "C#",
		"csharp":     "C#",
		"dotnet":     "C#",
		".net":       "C#",
		"ruby":       "Ruby",
		"rb":         "Ruby",
		"php":        "PHP",
		"scala":      "Scala",
		"elixir":     "Elixir",
		"erlang":     "Erlang",
		"haskell":    "Haskell",
		"clojure":    "Clojure",
		"clj":        "Clojure",
		"lua":        "Lua",
		"r":          "R",
		"rlang":      "R",
		"julia":      "Julia",
		"dart":       "Dart",
		"zig":        "Zig",
		"nim":        "Nim",
		"html":       "HTML",
		"html5":      "HTML",
		"css":        "CSS",
		"css3":       "CSS",
		"sql":        "SQL",
		"bash":       "Bash",
		"sh":         "Bash",
		"shell":      "Bash",
		"zsh":        "Bash",
		"powershell": "PowerShell",
		"pwsh":       "PowerShell",
	}
	for k, v := range defaultLangs {
		er.languages[k] = v
	}

	defaultFrameworks := map[string]string{
		"react":        "React",
		"reactjs":      "React",
		"react.js":     "React",
		"vue":          "Vue.js",
		"vuejs":        "Vue.js",
		"vue.js":       "Vue.js",
		"vue3":         "Vue.js",
		"angular":      "Angular",
		"angularjs":    "Angular",
		"svelte":       "Svelte",
		"sveltekit":    "Svelte",
		"nextjs":       "Next.js",
		"next.js":      "Next.js",
		"next":         "Next.js",
		"nuxt":         "Nuxt",
		"nuxtjs":       "Nuxt",
		"nuxt.js":      "Nuxt",
		"remix":        "Remix",
		"express":      "Express",
		"expressjs":    "Express",
		"express.js":   "Express",
		"fastify":      "Fastify",
		"nestjs":       "Nest.js",
		"nest.js":      "Nest.js",
		"nest":         "Nest.js",
		"nodejs":       "Node.js",
		"node.js":      "Node.js",
		"node":         "Node.js",
		"deno":         "Deno",
		"bun":          "Bun",
		"django":       "Django",
		"flask":        "Flask",
		"fastapi":      "FastAPI",
		"starlette":    "Starlette",
		"sqlalchemy":   "SQLAlchemy",
		"celery":       "Celery",
		"pandas":       "Pandas",
		"numpy":        "NumPy",
		"gin":          "Gin",
		"gin-gonic":    "Gin",
		"echo":         "Echo",
		"fiber":        "Fiber",
		"gofiber":      "Fiber",
		"chi":          "Chi",
		"go-chi":       "Chi",
		"gorm":         "GORM",
		"spring":       "Spring",
		"springboot":   "Spring",
		"rails":        "Rails",
		"ror":          "Rails",
		"sinatra":      "Sinatra",
		"actix":        "Actix",
		"actix-web":    "Actix",
		"axum":         "Axum",
		"rocket":       "Rocket",
		"tokio":        "Tokio",
		"react native": "React Native",
		"reactnative":  "React Native",
		"rn":           "React Native",
		"flutter":      "Flutter",
		"swiftui":      "SwiftUI",
		"tailwind":     "Tailwind CSS",
		"tailwindcss":  "Tailwind CSS",
		"bootstrap":    "Bootstrap",
		"mui":          "Material UI",
		"material ui":  "Material UI",
		"material-ui":  "Material UI",
		"jest":         "Jest",
		"vitest":       "Vitest",
		"pytest":       "Pytest",
		"cypress":      "Cypress",
		"playwright":   "Playwright",
	}
	for k, v := range defaultFrameworks {
		er.frameworks[k] = v
	}
}

func (er *EntityRecognizer) Recognize(terms []string) []RecognizedEntity {
	er.mu.RLock()
	defer er.mu.RUnlock()

	var entities []RecognizedEntity

	for i, term := range terms {
		termLower := strings.ToLower(term)

		if canonical, ok := er.products[termLower]; ok {
			entities = append(entities, RecognizedEntity{
				Text:      term,
				Type:      string(EntityTypeProduct),
				Canonical: canonical,
				Position:  i,
			})
			continue
		}

		if canonical, ok := er.organizations[termLower]; ok {
			entities = append(entities, RecognizedEntity{
				Text:      term,
				Type:      string(EntityTypeOrganization),
				Canonical: canonical,
				Position:  i,
			})
			continue
		}

		if canonical, ok := er.languages[termLower]; ok {
			entities = append(entities, RecognizedEntity{
				Text:      term,
				Type:      string(EntityTypeLanguage),
				Canonical: canonical,
				Position:  i,
			})
			continue
		}

		if canonical, ok := er.frameworks[termLower]; ok {
			entities = append(entities, RecognizedEntity{
				Text:      term,
				Type:      string(EntityTypeFramework),
				Canonical: canonical,
				Position:  i,
			})
		}
	}

	return entities
}

func (er *EntityRecognizer) RecognizeWithContext(query string) []RecognizedEntity {
	var entities []RecognizedEntity
	queryLower := strings.ToLower(query)

	er.mu.RLock()
	defer er.mu.RUnlock()

	for variant, canonical := range er.products {
		if strings.Contains(variant, " ") && strings.Contains(queryLower, variant) {
			entities = append(entities, RecognizedEntity{
				Text:      variant,
				Type:      string(EntityTypeProduct),
				Canonical: canonical,
				Position:  strings.Index(queryLower, variant),
			})
		}
	}

	for variant, canonical := range er.frameworks {
		if strings.Contains(variant, " ") && strings.Contains(queryLower, variant) {
			entities = append(entities, RecognizedEntity{
				Text:      variant,
				Type:      string(EntityTypeFramework),
				Canonical: canonical,
				Position:  strings.Index(queryLower, variant),
			})
		}
	}

	terms := strings.Fields(query)
	singleWordEntities := er.Recognize(terms)

	seen := make(map[string]bool)
	for _, e := range entities {
		seen[e.Canonical] = true
	}

	for _, e := range singleWordEntities {
		if !seen[e.Canonical] {
			entities = append(entities, e)
		}
	}

	return entities
}

func (er *EntityRecognizer) GetCanonical(term string) (string, EntityType) {
	er.mu.RLock()
	defer er.mu.RUnlock()

	termLower := strings.ToLower(term)

	if canonical, ok := er.products[termLower]; ok {
		return canonical, EntityTypeProduct
	}
	if canonical, ok := er.organizations[termLower]; ok {
		return canonical, EntityTypeOrganization
	}
	if canonical, ok := er.languages[termLower]; ok {
		return canonical, EntityTypeLanguage
	}
	if canonical, ok := er.frameworks[termLower]; ok {
		return canonical, EntityTypeFramework
	}

	return "", ""
}

func (er *EntityRecognizer) AddProduct(name string, variants []string) {
	er.mu.Lock()
	defer er.mu.Unlock()

	er.addEntity(er.products, EntityDefinition{Name: name, Variants: variants})
}

func (er *EntityRecognizer) AddOrganization(name string, variants []string) {
	er.mu.Lock()
	defer er.mu.Unlock()

	er.addEntity(er.organizations, EntityDefinition{Name: name, Variants: variants})
}

func (er *EntityRecognizer) AddLanguage(name string, variants []string) {
	er.mu.Lock()
	defer er.mu.Unlock()

	er.addEntity(er.languages, EntityDefinition{Name: name, Variants: variants})
}

func (er *EntityRecognizer) AddFramework(name string, variants []string) {
	er.mu.Lock()
	defer er.mu.Unlock()

	er.addEntity(er.frameworks, EntityDefinition{Name: name, Variants: variants})
}

func (er *EntityRecognizer) Stats() map[string]int {
	er.mu.RLock()
	defer er.mu.RUnlock()

	return map[string]int{
		"products":      len(er.products),
		"organizations": len(er.organizations),
		"languages":     len(er.languages),
		"frameworks":    len(er.frameworks),
	}
}
