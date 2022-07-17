module github.com/byzk-project-deploy/main-server

go 1.18

require (
	github.com/byzk-project-deploy/server-client-common v0.0.0-20220710124827-b36a9e32f8d5
	github.com/byzk-worker/go-common-logs v0.0.0-20220706092046-61e325f0c65c
	github.com/byzk-worker/go-db-utils v0.0.0-20220706170431-c3faa2b05109
	github.com/creack/pty v1.1.18
	github.com/desertbit/grumble v1.1.3
	github.com/devloperPlatform/go-coder-utils v0.0.0-20220303072916-d1e2b21739e1
	github.com/fsnotify/fsnotify v1.5.4
	github.com/gliderlabs/ssh v0.3.4
	github.com/jinzhu/gorm v1.9.16
	github.com/sirupsen/logrus v1.8.1
	github.com/sony/sonyflake v1.0.0
	github.com/spf13/viper v1.12.0
)

require (
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/desertbit/closer/v3 v3.1.2 // indirect
	github.com/desertbit/columnize v2.1.0+incompatible // indirect
	github.com/desertbit/go-shlex v0.1.1 // indirect
	github.com/desertbit/readline v1.5.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible // indirect
	github.com/lestrrat-go/strftime v1.0.6 // indirect
	github.com/m1/go-generate-password v0.2.0 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-sqlite3 v1.14.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.3.0 // indirect
	github.com/tjfoc/gmsm v1.4.0 // indirect
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4 // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	golang.org/x/text v0.3.7 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0 // indirect
)

replace github.com/byzk-project-deploy/server-client-common => ../server-client-common
