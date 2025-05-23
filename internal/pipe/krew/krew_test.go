package krew

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func createTemplateData() Manifest {
	return Manifest{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata: Metadata{
			Name: "Test",
		},
		Spec: Spec{
			Description:      "Some desc",
			Homepage:         "https://google.com",
			Version:          "v0.1.3",
			ShortDescription: "Short desc",
			Caveats:          "some caveat",
			Platforms: []Platform{
				{
					Selector: Selector{
						MatchLabels: MatchLabels{
							Arch: "amd64",
							Os:   "darwin",
						},
					},
					URI:    "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz",
					Sha256: "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68",
					Bin:    "test",
				},
				{
					Selector: Selector{
						MatchLabels: MatchLabels{
							Arch: "arm64",
							Os:   "darwin",
						},
					},
					URI:    "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_arm64.tar.gz",
					Sha256: "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68",
					Bin:    "test",
				},
				{
					Selector: Selector{
						MatchLabels: MatchLabels{
							Arch: "amd64",
							Os:   "linux",
						},
					},
					URI:    "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz",
					Sha256: "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
					Bin:    "test",
				},
				{
					Selector: Selector{
						MatchLabels: MatchLabels{
							Arch: "arm",
							Os:   "linux",
						},
					},
					URI:    "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Arm6.tar.gz",
					Sha256: "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
					Bin:    "test",
				},
				{
					Selector: Selector{
						MatchLabels: MatchLabels{
							Arch: "arm64",
							Os:   "linux",
						},
					},
					URI:    "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Arm64.tar.gz",
					Sha256: "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
					Bin:    "test",
				},
				{
					Selector: Selector{
						MatchLabels: MatchLabels{
							Arch: "amd64",
							Os:   "windows",
						},
					},
					URI:    "https://github.com/caarlos0/test/releases/download/v0.1.3/test_windows_amd64.zip",
					Sha256: "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
					Bin:    "test.exe",
				},
			},
		},
	}
}

func TestFullManifest(t *testing.T) {
	data := createTemplateData()
	data.Metadata.Name = manifestName(t)
	manifest, err := doBuildManifest(data)
	require.NoError(t, err)

	golden.RequireEqualYaml(t, []byte(manifest))
}

func TestSimple(t *testing.T) {
	data := createTemplateData()
	data.Metadata.Name = manifestName(t)
	manifest, err := doBuildManifest(data)
	require.NoError(t, err)
	golden.RequireEqualYaml(t, []byte(manifest))
}

func TestFullPipe(t *testing.T) {
	type testcase struct {
		prepare                func(ctx *context.Context)
		expectedRunError       string
		expectedRunErrorAs     any
		expectedPublishError   string
		expectedPublishErrorAs any
	}
	for name, tt := range map[string]testcase{
		"default": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Krews[0].Repository.Owner = "test"
				ctx.Config.Krews[0].Repository.Name = "test"
				ctx.Config.Krews[0].Homepage = "https://github.com/goreleaser"
			},
		},
		"git_remote": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Krews[0].Homepage = "https://github.com/goreleaser"
				ctx.Config.Krews[0].Repository = config.RepoRef{
					Name:   "test",
					Branch: "main",
					Git: config.GitRepoRef{
						URL:        testlib.GitMakeBareRepository(t),
						PrivateKey: testlib.MakeNewSSHKey(t, ""),
					},
				}
			},
		},
		"default_gitlab": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitLab
				ctx.Config.Krews[0].Repository.Owner = "test"
				ctx.Config.Krews[0].Repository.Name = "test"
				ctx.Config.Krews[0].Homepage = "https://gitlab.com/goreleaser"
			},
		},
		"invalid_commit_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Krews[0].Repository.Owner = "test"
				ctx.Config.Krews[0].Repository.Name = "test"
				ctx.Config.Krews[0].CommitMessageTemplate = "{{ .Asdsa }"
			},
			expectedPublishErrorAs: &tmpl.Error{},
		},
		"invalid desc": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Krews[0].Repository.Owner = "test"
				ctx.Config.Krews[0].Repository.Name = "test"
				ctx.Config.Krews[0].Description = "{{ .Asdsa }"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"invalid short desc": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Krews[0].Repository.Owner = "test"
				ctx.Config.Krews[0].Repository.Name = "test"
				ctx.Config.Krews[0].ShortDescription = "{{ .Asdsa }"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"invalid homepage": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Krews[0].Repository.Owner = "test"
				ctx.Config.Krews[0].Repository.Name = "test"
				ctx.Config.Krews[0].Homepage = "{{ .Asdsa }"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"invalid name": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Krews[0].Repository.Owner = "test"
				ctx.Config.Krews[0].Repository.Name = "test"
				ctx.Config.Krews[0].Name = "{{ .Asdsa }"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"invalid caveats": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Krews[0].Repository.Owner = "test"
				ctx.Config.Krews[0].Repository.Name = "test"
				ctx.Config.Krews[0].Caveats = "{{ .Asdsa }"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"no short desc": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Krews[0].Repository.Owner = "test"
				ctx.Config.Krews[0].Repository.Name = "test"
				ctx.Config.Krews[0].Description = "lalala"
				ctx.Config.Krews[0].ShortDescription = ""
			},
			expectedRunError: `krew: manifest short description is not set`,
		},
		"no desc": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Krews[0].Repository.Owner = "test"
				ctx.Config.Krews[0].Repository.Name = "test"
				ctx.Config.Krews[0].Description = ""
				ctx.Config.Krews[0].ShortDescription = "lalala"
			},
			expectedRunError: `krew: manifest description is not set`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()

			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: name,
					Krews: []config.Krew{
						{
							Name:             name,
							IDs:              []string{"foo"},
							Description:      "A run pipe test krew manifest and FOO={{ .Env.FOO }}",
							ShortDescription: "short desc {{.Env.BAR}}",
						},
					},
					Env: []string{"FOO=foo_is_bar", "BAR=honk"},
				},
				testctx.WithCurrentTag("v1.0.1"),
				testctx.WithVersion("1.0.1"),
			)
			tt.prepare(ctx)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "bar_bin.tar.gz",
				Path:    "doesnt matter",
				Goos:    "darwin",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:     "bar",
					artifact.ExtraFormat: "tar.gz",
				},
			})
			path := filepath.Join(folder, "bin.tar.gz")
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "bin.tar.gz",
				Path:    path,
				Goos:    "darwin",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:       "foo",
					artifact.ExtraFormat:   "tar.gz",
					artifact.ExtraBinaries: []string{"name"},
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "this-should-be-ignored.tar.gz",
				Path:    path,
				Goos:    "darwin",
				Goarch:  "amd64",
				Goamd64: "v3",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:       "foo",
					artifact.ExtraFormat:   "tar.gz",
					artifact.ExtraBinaries: []string{"name"},
				},
			})

			f, err := os.Create(path)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			client := client.NewMock()
			distFile := filepath.Join(folder, "krew", name+".yaml")

			require.NoError(t, Pipe{}.Default(ctx))

			err = runAll(ctx, client)
			if tt.expectedRunError != "" {
				require.EqualError(t, err, tt.expectedRunError)
				return
			}
			if tt.expectedRunErrorAs != nil {
				require.ErrorAs(t, err, tt.expectedRunErrorAs)
				return
			}
			require.NoError(t, err)

			err = publishAll(ctx, client)
			if tt.expectedPublishError != "" {
				require.EqualError(t, err, tt.expectedPublishError)
				return
			}
			if tt.expectedPublishErrorAs != nil {
				require.ErrorAs(t, err, tt.expectedPublishErrorAs)
				return
			}
			require.NoError(t, err)

			content := []byte(client.Content)
			if url := ctx.Config.Krews[0].Repository.Git.URL; url == "" {
				require.True(t, client.CreatedFile, "should have created a file")
			} else {
				content = testlib.CatFileFromBareRepositoryOnBranch(
					t, url,
					ctx.Config.Krews[0].Repository.Branch,
					"plugins/"+name+".yaml",
				)
			}

			golden.RequireEqualYaml(t, content)

			distBts, err := os.ReadFile(distFile)
			require.NoError(t, err)
			require.Equal(t, string(content), string(distBts))
		})
	}
}

func TestRunPipePullRequest(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Krews: []config.Krew{
				{
					Name:             "foo",
					Homepage:         "https://goreleaser.com",
					ShortDescription: "test",
					Description:      "Fake desc",
					Repository: config.RepoRef{
						Owner:  "foo",
						Name:   "bar",
						Branch: "update-{{.Version}}",
						PullRequest: config.PullRequest{
							Enabled: true,
							Base: config.PullRequestBase{
								Owner: "og",
								Name:  "bar",
							},
						},
					},
				},
			},
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
	)
	path := filepath.Join(folder, "dist/foo_darwin_all/foo")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_macos.tar.gz",
		Path:   path,
		Goos:   "darwin",
		Goarch: "all",
		Type:   artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	client := client.NewMock()
	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	require.True(t, client.OpenedPullRequest)
	require.True(t, client.SyncedFork)
	golden.RequireEqualYaml(t, []byte(client.Content))
}

func TestRunPipeUniversalBinary(t *testing.T) {
	folder := t.TempDir()

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "unibin",
			Krews: []config.Krew{
				{
					Name:             manifestName(t),
					Description:      "Some desc",
					ShortDescription: "Short desc",
					Repository: config.RepoRef{
						Owner: "unibin",
						Name:  "bar",
					},
					IDs: []string{
						"unibin",
					},
				},
			},
		},
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithVersion("1.0.1"),
	)

	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "unibin.tar.gz",
		Path:   path,
		Goos:   "darwin",
		Goarch: "all",
		Type:   artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "unibin",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"unibin"},
			artifact.ExtraReplaces: true,
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	client := client.NewMock()
	distFile := filepath.Join(folder, "krew", manifestName(t)+".yaml")

	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	golden.RequireEqualYaml(t, []byte(client.Content))
	distBts, err := os.ReadFile(distFile)
	require.NoError(t, err)
	require.Equal(t, client.Content, string(distBts))
}

func TestRunPipeUniversalBinaryNotReplacing(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "unibin",
			Krews: []config.Krew{
				{
					Name:             manifestName(t),
					Description:      "Some desc",
					ShortDescription: "Short desc",
					Repository: config.RepoRef{
						Owner: "unibin",
						Name:  "bar",
					},
					IDs: []string{
						"unibin",
					},
				},
			},
		},
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithVersion("1.0.1"),
	)
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "unibin_amd64.tar.gz",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "unibin",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"unibin"},
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "unibin_amd64.tar.gz",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "arm64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "unibin",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"unibin"},
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "unibin.tar.gz",
		Path:   path,
		Goos:   "darwin",
		Goarch: "all",
		Type:   artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "unibin",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"unibin"},
			artifact.ExtraReplaces: false,
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	client := client.NewMock()
	distFile := filepath.Join(folder, "krew", manifestName(t)+".yaml")

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	golden.RequireEqualYaml(t, []byte(client.Content))
	distBts, err := os.ReadFile(distFile)
	require.NoError(t, err)
	require.Equal(t, client.Content, string(distBts))
}

func TestRunPipeNameTemplate(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Krews: []config.Krew{
				{
					Name:             "{{ .Env.FOO_BAR }}",
					Description:      "Some desc",
					ShortDescription: "Short desc",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
				},
			},
			Env: []string{"FOO_BAR=" + t.Name()},
		},
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithVersion("1.0.1"),
	)
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin.tar.gz",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	client := client.NewMock()
	distFile := filepath.Join(folder, "krew", t.Name()+".yaml")

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	golden.RequireEqualYaml(t, []byte(client.Content))
	distBts, err := os.ReadFile(distFile)
	require.NoError(t, err)
	require.Equal(t, client.Content, string(distBts))
}

func TestRunPipeMultipleKrewWithSkip(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Krews: []config.Krew{
				{
					Name:             "foo",
					Description:      "Some desc",
					ShortDescription: "Short desc",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
					SkipUpload: "true",
				},
				{
					Name:             "bar",
					Description:      "Some desc",
					ShortDescription: "Short desc",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
				},
				{
					Name:             "foobar",
					Description:      "Some desc",
					ShortDescription: "Short desc",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
					SkipUpload: "true",
				},
			},
			Env: []string{"FOO_BAR=is_bar"},
		},
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithVersion("1.0.1"),
	)
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin.tar.gz",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cli := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, cli))
	require.EqualError(t, publishAll(ctx, cli), `krews.skip_upload is set`)
	require.True(t, cli.CreatedFile)

	for _, manifest := range ctx.Config.Krews {
		distFile := filepath.Join(folder, "krew", manifest.Name+".yaml")
		_, err := os.Stat(distFile)
		require.NoError(t, err, "file should exist: "+distFile)
	}
}

func TestRunPipeForMultipleArmVersions(t *testing.T) {
	for name, fn := range map[string]func(ctx *context.Context){
		"multiple_armv5": func(ctx *context.Context) {
			ctx.Config.Krews[0].Goarm = "5"
		},
		"multiple_armv6": func(ctx *context.Context) {
			ctx.Config.Krews[0].Goarm = "6"
		},
		"multiple_armv7": func(ctx *context.Context) {
			ctx.Config.Krews[0].Goarm = "7"
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: name,
					Krews: []config.Krew{
						{
							Name:             name,
							ShortDescription: "Short desc",
							Description:      "A run pipe test krew manifest and FOO={{ .Env.FOO }}",
							Repository: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Homepage: "https://github.com/goreleaser",
						},
					},
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Env: []string{"FOO=foo_is_bar"},
				},
				testctx.GitHubTokenType,
				testctx.WithVersion("1.0.1"),
				testctx.WithCurrentTag("v1.0.1"),
			)
			fn(ctx)
			for _, a := range []struct {
				name   string
				goos   string
				goarch string
				goarm  string
			}{
				{
					name:   "bin",
					goos:   "darwin",
					goarch: "amd64",
				},
				{
					name:   "arm64",
					goos:   "linux",
					goarch: "arm64",
				},
				{
					name:   "armv5",
					goos:   "linux",
					goarch: "arm",
					goarm:  "5",
				},
				{
					name:   "armv6",
					goos:   "linux",
					goarch: "arm",
					goarm:  "6",
				},
				{
					name:   "armv7",
					goos:   "linux",
					goarch: "arm",
					goarm:  "7",
				},
			} {
				path := filepath.Join(folder, fmt.Sprintf("%s.tar.gz", a.name))
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:    fmt.Sprintf("%s.tar.gz", a.name),
					Path:    path,
					Goos:    a.goos,
					Goarch:  a.goarch,
					Goarm:   a.goarm,
					Goamd64: "v1",
					Type:    artifact.UploadableArchive,
					Extra: map[string]any{
						artifact.ExtraID:       a.name,
						artifact.ExtraFormat:   "tar.gz",
						artifact.ExtraBinaries: []string{"foo"},
					},
				})
				f, err := os.Create(path)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			client := client.NewMock()
			distFile := filepath.Join(folder, "krew", name+".yaml")

			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, runAll(ctx, client))
			require.NoError(t, publishAll(ctx, client))
			require.True(t, client.CreatedFile)
			golden.RequireEqualYaml(t, []byte(client.Content))

			distBts, err := os.ReadFile(distFile)
			require.NoError(t, err)
			require.Equal(t, client.Content, string(distBts))
		})
	}
}

func TestRunPipeNoBuilds(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Krews: []config.Krew{
			{
				Name:             manifestName(t),
				Description:      "Some desc",
				ShortDescription: "Short desc",
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	}, testctx.GitHubTokenType)
	client := client.NewMock()
	require.Equal(t, ErrNoArchivesFound, runAll(ctx, client))
	require.False(t, client.CreatedFile)
}

func TestRunPipeNoUpload(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Krews: []config.Krew{
			{
				Name:             manifestName(t),
				Description:      "Some desc",
				ShortDescription: "Short desc",
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	}, testctx.GitHubTokenType, testctx.WithCurrentTag("v1.0.1"))
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})
	client := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))

	assertNoPublish := func(t *testing.T) {
		t.Helper()
		require.NoError(t, runAll(ctx, client))
		testlib.AssertSkipped(t, publishAll(ctx, client))
		require.False(t, client.CreatedFile)
	}
	t.Run("skip upload true", func(t *testing.T) {
		ctx.Config.Krews[0].SkipUpload = "true"
		ctx.Semver.Prerelease = ""
		assertNoPublish(t)
	})
	t.Run("skip upload auto", func(t *testing.T) {
		ctx.Config.Krews[0].SkipUpload = "auto"
		ctx.Semver.Prerelease = "beta1"
		assertNoPublish(t)
	})
}

func TestRunEmptyTokenType(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Krews: []config.Krew{
			{
				Name:             manifestName(t),
				Description:      "Some desc",
				ShortDescription: "Short desc",
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	}, testctx.WithCurrentTag("v1.0.1"))
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"bin"},
		},
	})
	client := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, client))
}

func TestRunMultipleBinaries(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Krews: []config.Krew{
			{
				Name:             manifestName(t),
				Description:      "Some desc",
				ShortDescription: "Short desc",
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	}, testctx.WithCurrentTag("v1.0.1"))
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin.tar.gz",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"bin1", "bin2"},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	client := client.NewMock()
	require.EqualError(t, runAll(ctx, client), `krew: only one binary per archive allowed, got 2 on "bin.tar.gz"`)
}

func TestDefault(t *testing.T) {
	testlib.Mktmp(t)

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "myproject",
		Krews: []config.Krew{
			{
				Repository: config.RepoRef{
					Git: config.GitRepoRef{
						URL: "foo/bar",
					},
				},
			},
		},
	}, testctx.GitHubTokenType)
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Krews[0].Name)
	require.NotEmpty(t, ctx.Config.Krews[0].CommitAuthor.Name)
	require.NotEmpty(t, ctx.Config.Krews[0].CommitAuthor.Email)
	require.NotEmpty(t, ctx.Config.Krews[0].CommitMessageTemplate)
	require.Equal(t, "foo/bar", ctx.Config.Krews[0].Repository.Git.URL)
}

func TestGHFolder(t *testing.T) {
	require.Equal(t, "bar.yaml", buildManifestPath("", "bar.yaml"))
	require.Equal(t, "fooo/bar.yaml", buildManifestPath("fooo", "bar.yaml"))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Krews: []config.Krew{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRunSkipNoName(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Krews: []config.Krew{{}},
	})

	client := client.NewMock()
	testlib.AssertSkipped(t, runAll(ctx, client))
}

func manifestName(tb testing.TB) string {
	tb.Helper()
	return path.Base(tb.Name())
}
