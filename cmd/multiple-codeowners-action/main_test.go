package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteCodeownersRules(t *testing.T) {
	tests := []struct {
		name            string
		setupFiles      map[string]string // Maps file paths to contents
		expectedRewrite []string
	}{
		{
			name: "simple structure",
			setupFiles: map[string]string{
				"CODEOWNERS":               "* @org/main-owner",
				"src/CODEOWNERS":           "@org/src-owner",
				"src/api/CODEOWNERS":       "@org/api-owner",
				"src/api/utils/CODEOWNERS": "@org/utils-owner",
			},
			expectedRewrite: []string{
				"* @org/main-owner",
				"/src @org/src-owner",
				"/src/api @org/api-owner",
				"/src/api/utils @org/utils-owner",
			},
		},
		{
			name: "complex structure with patterns",
			setupFiles: map[string]string{
				"CODEOWNERS":               "* @org/main-owner",
				"src/CODEOWNERS":           "*.go @org/go-owner",
				"src/api/CODEOWNERS":       "*.py @org/python-owner",
				"src/api/utils/CODEOWNERS": "utils.go @org/utils-owner",
				"docs/CODEOWNERS":          "* @org/docs-owner",
			},
			expectedRewrite: []string{
				"* @org/main-owner",
				"/docs/* @org/docs-owner",
				"/src/*.go @org/go-owner",
				"/src/api/*.py @org/python-owner",
				"/src/api/utils/utils.go @org/utils-owner",
			},
		},
		{
			name: "empty CODEOWNERS files",
			setupFiles: map[string]string{
				"CODEOWNERS":         "",
				"src/CODEOWNERS":     "",
				"src/api/CODEOWNERS": "",
			},
			expectedRewrite: nil,
		},
		{
			name: "nested structure with similar subpaths",
			setupFiles: map[string]string{
				"CODEOWNERS":                "* @org/main-owner",
				"src/CODEOWNERS":            "@org/src-owner",
				"src/api/CODEOWNERS":        "@org/api-owner",
				"src/api2/CODEOWNERS":       "@org/api2-owner",
				"src/api2/utils/CODEOWNERS": "@org/api2-utils-owner",
			},
			expectedRewrite: []string{
				"* @org/main-owner",
				"/src @org/src-owner",
				"/src/api @org/api-owner",
				"/src/api2 @org/api2-owner",
				"/src/api2/utils @org/api2-utils-owner",
			},
		},
		{
			name: "overlapping patterns",
			setupFiles: map[string]string{
				"CODEOWNERS":           "* @org/main-owner",
				"src/CODEOWNERS":       "*.go @org/go-owner \n*.js @org/js-owner",
				"src/utils/CODEOWNERS": "*.go @org/utils-go-owner",
			},
			expectedRewrite: []string{
				"* @org/main-owner",
				"/src/*.go @org/go-owner",
				"/src/*.js @org/js-owner",
				"/src/utils/*.go @org/utils-go-owner",
			},
		},
		{
			name: "CODEOWNERS in hidden directories",
			setupFiles: map[string]string{
				".foo/CODEOWNERS":           "* @org/gh-owner",
				".foo/workflows/CODEOWNERS": "* @org/gh-wf-owner",
			},
			expectedRewrite: []string{
				"/.foo/* @org/gh-owner",
				"/.foo/workflows/* @org/gh-wf-owner",
			},
		},
		{
			name: "CODEOWNERS with special characters in paths",
			setupFiles: map[string]string{
				"CODEOWNERS":         "* @org/main-owner",
				"src-foo/CODEOWNERS": "@org/src-foo-owner",
				"src_bar/CODEOWNERS": "@org/src_bar-owner",
			},
			expectedRewrite: []string{
				"* @org/main-owner",
				"/src-foo @org/src-foo-owner",
				"/src_bar @org/src_bar-owner",
			},
		},
		{
			name: "single path with multiple code owners",
			setupFiles: map[string]string{
				"CODEOWNERS":     "* @org/main-owner",
				"src/CODEOWNERS": "main.go @org/dev1 @org/dev2 @org/dev3",
			},
			expectedRewrite: []string{
				"* @org/main-owner",
				"/src/main.go @org/dev1 @org/dev2 @org/dev3",
			},
		},
		{
			name: "multiple paths with multiple code owners",
			setupFiles: map[string]string{
				"CODEOWNERS":      "* @org/main-owner",
				"src/CODEOWNERS":  "main.go @org/dev1 @org/dev2",
				"docs/CODEOWNERS": "README.md @org/doc1 @org/doc2 @org/doc3",
			},
			expectedRewrite: []string{
				"* @org/main-owner",
				"/docs/README.md @org/doc1 @org/doc2 @org/doc3",
				"/src/main.go @org/dev1 @org/dev2",
			},
		},
		{
			name: "complex patterns with multiple code owners",
			setupFiles: map[string]string{
				"CODEOWNERS":               "* @org/main-owner",
				"src/CODEOWNERS":           "*.go @org/go-dev1 @org/go-dev2",
				"src/api/CODEOWNERS":       "*.py @org/py-dev",
				"src/api/utils/CODEOWNERS": "*.ts @org/ts-dev1 @org/ts-dev2",
				"src/utils/CODEOWNERS":     "*.js @org/js-dev1 @org/js-dev2 @org/js-dev3",
			},
			expectedRewrite: []string{
				"* @org/main-owner",
				"/src/*.go @org/go-dev1 @org/go-dev2",
				"/src/api/*.py @org/py-dev",
				"/src/utils/*.js @org/js-dev1 @org/js-dev2 @org/js-dev3",
				"/src/api/utils/*.ts @org/ts-dev1 @org/ts-dev2",
			},
		},
		{
			name: "paths with overlapping and multiple code owners",
			setupFiles: map[string]string{
				"CODEOWNERS":                   "* @org/main-owner",
				"src/CODEOWNERS":               "*.go @org/go-owner1 \n*.js @org/js-owner1 @org/js-owner2",
				"src/utils/CODEOWNERS":         "*.go @org/utils-go-owner1 @org/utils-go-owner2",
				"src/utils/special/CODEOWNERS": "special.go @org/special-go-owner1 @org/special-go-owner2",
			},
			expectedRewrite: []string{
				"* @org/main-owner",
				"/src/*.go @org/go-owner1",
				"/src/*.js @org/js-owner1 @org/js-owner2",
				"/src/utils/*.go @org/utils-go-owner1 @org/utils-go-owner2",
				"/src/utils/special/special.go @org/special-go-owner1 @org/special-go-owner2",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir, err := os.MkdirTemp("", "test-rewrite-codeowners")
			require.NoError(t, err)
			defer os.RemoveAll(tmpdir)

			// Setup test repository structure
			for path, content := range tc.setupFiles {
				fullPath := filepath.Join(tmpdir, path)
				err = os.MkdirAll(filepath.Dir(fullPath), 0o700)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0o600)
				require.NoError(t, err)
			}

			// Test RewriteCodeownersRules
			args.Output = ".github/CODEOWNERS"
			rewrittenRules, err := RewriteCodeownersRules(tmpdir)
			require.NoError(t, err)
			require.Equal(t, tc.expectedRewrite, rewrittenRules)
		})
	}
}

func TestIsDirRule(t *testing.T) {
	assert.True(t, isDirRule("@owner"))
	assert.False(t, isDirRule("file.txt @owner"))
}

func TestRewriteDirRule(t *testing.T) {
	assert.Equal(t, "* @owner", rewriteDirRule("*", "@owner"))
	assert.Equal(t, "/src @owner", rewriteDirRule("/src", "@owner"))
}

func TestRewriteNonDirRule(t *testing.T) {
	assert.Equal(t, "/src/file.txt @owner", rewriteNonDirRule("/src", "file.txt @owner"))
}

func TestGenerateCodeownersFile(t *testing.T) {
	tests := []struct {
		name     string
		rules    []string
		expected string
	}{
		{
			name:     "empty rules",
			rules:    []string{},
			expected: GeneratedFileWarning + "\n\n\n",
		},
		{
			name:  "single team rule",
			rules: []string{"* @team/backend"},
			expected: GeneratedFileWarning + `

* @team/backend
`,
		},
		{
			name: "mixed team and individual rules",
			rules: []string{
				"/api @team/api-team",
				"/frontend @team/frontend",
				"/README.md @john.doe",
			},
			expected: GeneratedFileWarning + `

/api @team/api-team
/frontend @team/frontend
/README.md @john.doe
`,
		},
		{
			name: "complex rules with comments",
			rules: []string{
				"# API ownership",
				"/api @team/api-team",
				"# Frontend ownership",
				"/frontend @team/frontend",
			},
			expected: GeneratedFileWarning + `

# API ownership
/api @team/api-team
# Frontend ownership
/frontend @team/frontend
`,
		},
		{
			name: "glob patterns with teams",
			rules: []string{
				"*.go @team/go-developers",
				"*.js @team/js-developers",
			},
			expected: GeneratedFileWarning + `

*.go @team/go-developers
*.js @team/js-developers
`,
		},
		{
			name: "nested directories with individuals",
			rules: []string{
				"/services/auth @alice",
				"/services/payment @bob",
			},
			expected: GeneratedFileWarning + `

/services/auth @alice
/services/payment @bob
`,
		},
		{
			name: "complex rules with glob patterns",
			rules: []string{
				"/services @team/services",
				"/services/*.go @team/go-services",
				"/services/*.py @team/python-services",
			},
			expected: GeneratedFileWarning + `

/services @team/services
/services/*.go @team/go-services
/services/*.py @team/python-services
`,
		},
		{
			name: "real world example with mixed handles",
			rules: []string{
				"/ @team/admin",
				"/src @team/developers",
				"/docs @team/documentation",
				"/test @team/testing",
				"/deploy.yaml @alice",
			},
			expected: GeneratedFileWarning + `

/ @team/admin
/src @team/developers
/docs @team/documentation
/test @team/testing
/deploy.yaml @alice
`,
		},
		{
			name: "rules with special characters",
			rules: []string{
				"/src#old @team/legacy-code",
				"/new-feature-branch* @team/new-feature",
			},
			expected: GeneratedFileWarning + `

/src#old @team/legacy-code
/new-feature-branch* @team/new-feature
`,
		},
		{
			name: "rules with subdirectories and multiple owners",
			rules: []string{
				"/src/utils @team/utility @alice",
				"/src/api @team/api @bob @charlie",
			},
			expected: GeneratedFileWarning + `

/src/utils @team/utility @alice
/src/api @team/api @bob @charlie
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := GenerateCodeownersFile(tc.rules)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestQueue(t *testing.T) {
	queue := NewQueue[int]()

	// Test for an empty queue
	assert.Equal(t, 0, queue.Len(), "Queue length should be 0 on initialization")
	assert.Equal(t, 0, queue.Dequeue(), "Dequeue on empty queue should return zero value")

	// Test enqueue and length
	queue.Enqueue(1)
	queue.Enqueue(2)
	queue.Enqueue(3)
	assert.Equal(t, 3, queue.Len(), "Queue length should be 3 after enqueuing 3 items")

	// Test dequeue and length
	assert.Equal(t, 1, queue.Dequeue(), "Dequeue should return the first item")
	assert.Equal(t, 2, queue.Len(), "Queue length should be 2 after dequeuing an item")

	assert.Equal(t, 2, queue.Dequeue(), "Dequeue should return the second item")
	assert.Equal(t, 1, queue.Len(), "Queue length should be 1 after dequeuing a second item")

	assert.Equal(t, 3, queue.Dequeue(), "Dequeue should return the third item")
	assert.Equal(t, 0, queue.Len(), "Queue length should be 0 after dequeuing a third item")

	// Ensure queue is empty again
	assert.Equal(t, 0, queue.Dequeue(), "Dequeue on empty queue should return zero value")
	assert.Equal(t, 0, queue.Len(), "Queue length should be 0 after dequeuing all items")
}
