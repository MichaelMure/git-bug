module github.com/MichaelMure/git-bug

go 1.22.5

// https://github.com/praetorian-inc/gokart/pull/84
replace github.com/praetorian-inc/gokart v0.5.1 => github.com/selesy/gokart v0.5.2-rc1

// caused by github.com/blevesearch/bleve@v1.0.14
// caused by github.com/couchbase/vellum@v1.0.2
replace github.com/willf/bitset v1.1.11 => github.com/bits-and-blooms/bitset v1.1.11

require (
	github.com/99designs/gqlgen v0.17.49
	github.com/99designs/keyring v1.2.2
	github.com/MichaelMure/go-term-text v0.3.1
	github.com/ProtonMail/go-crypto v1.0.0
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/awesome-gocui/gocui v1.1.0
	github.com/blevesearch/bleve v1.0.14
	github.com/cheekybits/genny v1.0.0
	github.com/dustin/go-humanize v1.0.1
	github.com/fatih/color v1.17.0
	github.com/go-git/go-billy/v5 v5.5.0
	github.com/go-git/go-git/v5 v5.12.0
	github.com/gorilla/mux v1.8.1
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/icrowley/fake v0.0.0-20240710202011-f797eb4a99c0
	github.com/mattn/go-isatty v0.0.20
	github.com/phayes/freeport v0.0.0-20220201140144-74d24b5ae9f5
	github.com/pkg/errors v0.9.1
	github.com/shurcooL/githubv4 v0.0.0-20240429030203-be2daab69064
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.9.0
	github.com/vbauerster/mpb/v8 v8.7.5
	github.com/vektah/gqlparser/v2 v2.5.16
	github.com/xanzy/go-gitlab v0.107.0
	golang.org/x/crypto v0.26.0
	golang.org/x/oauth2 v0.22.0
	golang.org/x/sync v0.8.0
	golang.org/x/sys v0.24.0
	golang.org/x/text v0.17.0
)

require (
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/bits-and-blooms/bitset v1.13.0 // indirect
	github.com/cyphar/filepath-securejoin v0.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/sosodev/duration v1.3.1 // indirect
	golang.org/x/telemetry v0.0.0-20240723021908-ccdfb411a0c4 // indirect
)

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/RoaringBitmap/roaring v1.9.4 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/mmap-go v1.0.4 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/zap/v11 v11.0.14 // indirect
	github.com/blevesearch/zap/v12 v12.0.14 // indirect
	github.com/blevesearch/zap/v13 v13.0.6 // indirect
	github.com/blevesearch/zap/v14 v14.0.5 // indirect
	github.com/blevesearch/zap/v15 v15.0.3 // indirect
	github.com/cloudflare/circl v1.3.9 // indirect
	github.com/corpix/uarand v0.2.0 // indirect
	github.com/couchbase/vellum v1.0.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/danieljoos/wincred v1.2.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dvsekhvalnov/jose2go v1.7.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/gdamore/tcell/v2 v2.7.4 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/lithammer/dedent v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/owenrumney/go-sarif v1.1.1 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/praetorian-inc/gokart v0.5.1
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/shurcooL/graphql v0.0.0-20230722043721-ed46e5a46466 // indirect
	github.com/skeema/knownhosts v1.3.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/steveyen/gtreap v0.1.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/urfave/cli/v2 v2.27.2 // indirect
	github.com/willf/bitset v1.1.11 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/zclconf/go-cty v1.15.0 // indirect
	go.etcd.io/bbolt v1.3.10 // indirect
	golang.org/x/mod v0.19.0 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/term v0.23.0
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.23.0 // indirect
	golang.org/x/vuln v1.1.3
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
