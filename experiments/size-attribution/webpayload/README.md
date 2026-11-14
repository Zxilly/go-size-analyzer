# Web payload projection comparison

This harness uses a prior GSA CLI to generate full JSON and HTML in one analysis,
then feeds that exact full report to the current serializer. Analyzer changes do
not affect the measured transport difference. Original binaries and baseline
executables remain untouched.

```powershell
python experiments/size-attribution/webpayload/run.py --baseline-cli temp/size-iteration/gsa-embedded.exe
```

The default matrix is the Go 1.23 Linux example, Docker Compose and
analysis-server from `scripts/bins`, with and without `--coverage-details`.
Override samples with repeated `--sample LABEL=PATH` arguments. Set
`--output-dir` to choose an artifact directory; the default is
`temp/size-iteration2/transport`. `--compare-only` validates already-generated
artifacts without rebuilding or rerunning analyses.

The harness builds only its projector with `-mod=readonly -p=2 -tags=embed` and
uses `GOMAXPROCS=4`. It requires the repository's embedded UI template to be
available. No timing or memory performance claim is inferred from these runs.

`summary.json` measures UTF-8 bytes of the embedded JSON data. HTML templates may
differ between CLI versions, so total HTML size is recorded separately and
`template_equal` makes that distinction explicit. Assertions verify:

- All functions, metrics, known-zero/unknown references and sampled evidence survive.
- Removed function component maps/ranges are replaced by the same pcln total.
- Coverage counts, sources and notes survive; only the unused web region list is omitted.
- Full CLI JSON roundtrips without changes, and compact JSON still retains requested regions.
- The HTML data is the same as direct serialization through the web projector.

Native-versus-WASM equivalence and writer failures are covered by the production
marshaler/result/printer tests, rather than inferred from the byte counts.
