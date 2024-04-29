module github.com/Zxilly/go-size-analyzer

go 1.22

toolchain go1.22.1

require (
	github.com/alecthomas/kong v0.9.0
	github.com/deckarep/golang-set/v2 v2.6.0
	github.com/dghubble/trie v0.1.0
	github.com/dustin/go-humanize v1.0.1
	github.com/goretk/gore v0.11.5
	github.com/jedib0t/go-pretty/v6 v6.5.8
	github.com/nikolaydubina/treemap v1.2.5
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/samber/lo v1.39.0
	github.com/stretchr/testify v1.9.0
	go4.org/intern v0.0.0-20230525184215-6c62f75575cb
	golang.org/x/arch v0.7.0
	golang.org/x/exp v0.0.0-20240416160154-fe59bbe5cc7f
	golang.org/x/net v0.24.0
)

replace (
	github.com/dghubble/trie v0.1.0 => github.com/ZxillyFork/trie v0.0.0-20240428062955-77f35217e179
	github.com/goretk/gore v0.11.5 => github.com/Zxilly/gore v0.0.0-20240422132935-dedfb5d7e0cf
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20231121144256-b99613f794b6 // indirect
	golang.org/x/sys v0.19.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
