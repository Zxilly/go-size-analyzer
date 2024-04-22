module github.com/Zxilly/go-size-analyzer

go 1.22

toolchain go1.22.1

require (
	github.com/ZxillyFork/go-flags v0.0.0-20240325132113-057f93e1e1ff
	github.com/deckarep/golang-set/v2 v2.6.0
	github.com/dustin/go-humanize v1.0.1
	github.com/goretk/gore v0.11.5
	github.com/jedib0t/go-pretty/v6 v6.5.5
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/samber/lo v1.39.0
	github.com/schollz/progressbar/v3 v3.14.2
	go4.org/intern v0.0.0-20230525184215-6c62f75575cb
	golang.org/x/arch v0.7.0
	golang.org/x/exp v0.0.0-20240416160154-fe59bbe5cc7f
)

require (
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20231121144256-b99613f794b6 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/term v0.18.0 // indirect
)

replace github.com/goretk/gore v0.11.5 => github.com/Zxilly/gore v0.0.0-20240422132935-dedfb5d7e0cf
