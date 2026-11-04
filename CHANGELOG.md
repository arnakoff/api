# Changelog

## [0.56.2](https://github.com/arnakoff/api/compare/v0.56.1...v0.56.2) (2026-11-04)


### Bug Fixes

* **ci:** leave the opentofu state bucket to the owner ([#290](https://github.com/arnakoff/api/issues/290)) ([43f84ac](https://github.com/arnakoff/api/commit/43f84acb97c0bc8f3b76af99d571b94d6e27fa0d)), closes [#289](https://github.com/arnakoff/api/issues/289)
* **httpapi:** spend one unit of the key quota per batch item ([#292](https://github.com/arnakoff/api/issues/292)) ([72eed01](https://github.com/arnakoff/api/commit/72eed0154c791eb0749acc4b93001dc355c3766b)), closes [#291](https://github.com/arnakoff/api/issues/291)

## [0.56.1](https://github.com/arnakoff/api/compare/v0.56.0...v0.56.1) (2026-11-04)


### Bug Fixes

* **platform:** start no registry lookup for a check past its deadline ([#288](https://github.com/arnakoff/api/issues/288)) ([a4fb77c](https://github.com/arnakoff/api/commit/a4fb77cdf544cea715fa57f466f70ad488fd16c9)), closes [#287](https://github.com/arnakoff/api/issues/287)
* **sync:** compare the shrink against the stored snapshot when none is in service ([#283](https://github.com/arnakoff/api/issues/283)) ([bb05d01](https://github.com/arnakoff/api/commit/bb05d01e3a5cd86a260ec7cb807689f1ff367313)), closes [#282](https://github.com/arnakoff/api/issues/282)
* **sync:** refuse a snapshot whose rows the readers cannot index ([#286](https://github.com/arnakoff/api/issues/286)) ([3cbf866](https://github.com/arnakoff/api/commit/3cbf866a8ba978af3ddfa2773b4e59291c42f47e)), closes [#285](https://github.com/arnakoff/api/issues/285)

## [0.56.0](https://github.com/arnakoff/api/compare/v0.55.0...v0.56.0) (2026-11-04)


### Features

* **ci:** publish the mcp server to the official registry on each release ([#279](https://github.com/arnakoff/api/issues/279)) ([cf5b4ee](https://github.com/arnakoff/api/commit/cf5b4ee24ed32608ff6248c186f063967b426447)), closes [#278](https://github.com/arnakoff/api/issues/278)

## [0.55.0](https://github.com/arnakoff/api/compare/v0.54.0...v0.55.0) (2026-11-04)


### Features

* **mcp:** describe the remote server for the official mcp registry ([#275](https://github.com/arnakoff/api/issues/275)) ([3d6b303](https://github.com/arnakoff/api/commit/3d6b303771763c2f2dd8b26dfba93659ce3d6c4a))

## [0.54.0](https://github.com/arnakoff/api/compare/v0.53.1...v0.54.0) (2026-11-04)


### Features

* **ci:** alert on sync failures and probe production from outside ([#271](https://github.com/arnakoff/api/issues/271)) ([9552f53](https://github.com/arnakoff/api/commit/9552f532c7bce9d596119d7c9a00ef4a973406b0)), closes [#72](https://github.com/arnakoff/api/issues/72)


### Bug Fixes

* **ci:** keep the api.arnakoff.com domain when an apply omits api_hostname ([#274](https://github.com/arnakoff/api/issues/274)) ([9a37bd8](https://github.com/arnakoff/api/commit/9a37bd889751efa2503b6db76b853bc73dd421b0)), closes [#273](https://github.com/arnakoff/api/issues/273)

## [0.53.1](https://github.com/arnakoff/api/compare/v0.53.0...v0.53.1) (2026-11-04)


### Bug Fixes

* **ci:** let the admin role connect to the arnakoff database ([#262](https://github.com/arnakoff/api/issues/262)) ([d044bed](https://github.com/arnakoff/api/commit/d044bede49ea18642fcbc620072a851897b2bbf9)), closes [#261](https://github.com/arnakoff/api/issues/261)
* **ci:** let the deploy workflow create the sync job's secret versions ([#265](https://github.com/arnakoff/api/issues/265)) ([7807f42](https://github.com/arnakoff/api/commit/7807f426a63117f7f394d9570eb5481162f8fa3e)), closes [#264](https://github.com/arnakoff/api/issues/264)

## [0.53.0](https://github.com/arnakoff/api/compare/v0.52.0...v0.53.0) (2026-11-04)


### Features

* **ci:** apply the migrations from the private network ([#260](https://github.com/arnakoff/api/issues/260)) ([4ea3086](https://github.com/arnakoff/api/commit/4ea30861afb8b4a907286da127d9cfff7f36a961)), closes [#259](https://github.com/arnakoff/api/issues/259)
* **ci:** deploy signed images to scaleway, infrastructure in opentofu ([#251](https://github.com/arnakoff/api/issues/251)) ([d8489fc](https://github.com/arnakoff/api/commit/d8489fc24d08759e96d9e2778356e03579d4d4bb)), closes [#71](https://github.com/arnakoff/api/issues/71)


### Bug Fixes

* **ci:** give scw and the database url outputs they can use ([#258](https://github.com/arnakoff/api/issues/258)) ([0960e83](https://github.com/arnakoff/api/commit/0960e83abdd689312b3af23e65bebbf5b4b14f1a)), closes [#257](https://github.com/arnakoff/api/issues/257)
* **ci:** let the first production apply and deploy go through ([#256](https://github.com/arnakoff/api/issues/256)) ([a40e823](https://github.com/arnakoff/api/commit/a40e8230289d6fa4447242dea0d257e18e430460)), closes [#255](https://github.com/arnakoff/api/issues/255)

## [0.52.0](https://github.com/arnakoff/api/compare/v0.51.0...v0.52.0) (2026-11-04)


### Features

* **httpapi:** issue free api keys by email login link ([#246](https://github.com/arnakoff/api/issues/246)) ([b46d0af](https://github.com/arnakoff/api/commit/b46d0af9fe601f1f3ead7d95b33cc1cc2ef0e46b)), closes [#59](https://github.com/arnakoff/api/issues/59)
* **httpapi:** read the client address from a field the platform overwrites ([#248](https://github.com/arnakoff/api/issues/248)) ([0b83973](https://github.com/arnakoff/api/commit/0b83973ca8f21bc736d5f1c5e41142426683c87c)), closes [#247](https://github.com/arnakoff/api/issues/247)
* **store:** verify the database against a certificate authority given in pem ([#250](https://github.com/arnakoff/api/issues/250)) ([62602c5](https://github.com/arnakoff/api/commit/62602c54ac80537ec2d9ae3d414a216219da6176)), closes [#249](https://github.com/arnakoff/api/issues/249)

## [0.51.0](https://github.com/arnakoff/api/compare/v0.50.0...v0.51.0) (2026-11-03)


### Features

* **app:** purge api keys revoked more than a year ago ([#243](https://github.com/arnakoff/api/issues/243)) ([309e73f](https://github.com/arnakoff/api/commit/309e73f85952481c44c5da4969f457947cde468c)), closes [#73](https://github.com/arnakoff/api/issues/73)

## [0.50.0](https://github.com/arnakoff/api/compare/v0.49.0...v0.50.0) (2026-11-03)


### Features

* **sync:** store snapshots in scaleway object storage in production ([#241](https://github.com/arnakoff/api/issues/241)) ([b336f81](https://github.com/arnakoff/api/commit/b336f81d86310458e35dd83ee8bb7676107f37ed)), closes [#236](https://github.com/arnakoff/api/issues/236)

## [0.49.0](https://github.com/arnakoff/api/compare/v0.48.0...v0.49.0) (2026-11-03)


### Features

* **ci:** publish signed images to ghcr with sbom and provenance ([#231](https://github.com/arnakoff/api/issues/231)) ([faf4880](https://github.com/arnakoff/api/commit/faf4880212d56daad73877be582b7e85de8741cf)), closes [#69](https://github.com/arnakoff/api/issues/69)

## [0.48.0](https://github.com/arnakoff/api/compare/v0.47.1...v0.48.0) (2026-11-03)


### Features

* build a distroless non-root container image ([#229](https://github.com/arnakoff/api/issues/229)) ([71b2d04](https://github.com/arnakoff/api/commit/71b2d04f46230daa729a78a3795dc7486c9b6c4b)), closes [#68](https://github.com/arnakoff/api/issues/68)

## [0.47.1](https://github.com/arnakoff/api/compare/v0.47.0...v0.47.1) (2026-11-03)


### Bug Fixes

* **domain:** quote third-party text without hidden characters ([#227](https://github.com/arnakoff/api/issues/227)) ([01eacd0](https://github.com/arnakoff/api/commit/01eacd0926f7f75f951fcb9362b1d9e76b707ac3)), closes [#65](https://github.com/arnakoff/api/issues/65)

## [0.47.0](https://github.com/arnakoff/api/compare/v0.46.0...v0.47.0) (2026-11-03)


### Features

* **mcp:** serve the MCP tools over HTTP behind the API key ([#225](https://github.com/arnakoff/api/issues/225)) ([1b8b6a9](https://github.com/arnakoff/api/commit/1b8b6a92ac91df4726af185f25abf26187411032)), closes [#64](https://github.com/arnakoff/api/issues/64)

## [0.46.0](https://github.com/arnakoff/api/compare/v0.45.0...v0.46.0) (2026-11-03)


### Features

* **httpapi:** declare the public host in the contract ([#223](https://github.com/arnakoff/api/issues/223)) ([51702b0](https://github.com/arnakoff/api/commit/51702b05cce18ca0d330b6d766b2ee719e73313e)), closes [#187](https://github.com/arnakoff/api/issues/187)
* **httpapi:** load the official domains in every check ([#221](https://github.com/arnakoff/api/issues/221)) ([ad7dfba](https://github.com/arnakoff/api/commit/ad7dfba455c25c23af19cd17477bdebca3311305)), closes [#167](https://github.com/arnakoff/api/issues/167)

## [0.45.0](https://github.com/arnakoff/api/compare/v0.44.1...v0.45.0) (2026-11-03)


### Features

* **mcp:** serve check and list_sources over stdio ([#217](https://github.com/arnakoff/api/issues/217)) ([5066d6e](https://github.com/arnakoff/api/commit/5066d6eac17bccd0051a36b834d6b685d5e0e619)), closes [#63](https://github.com/arnakoff/api/issues/63)

## [0.44.1](https://github.com/arnakoff/api/compare/v0.44.0...v0.44.1) (2026-11-03)


### Bug Fixes

* **httpapi:** spend the refused-key budget only on refusals ([#215](https://github.com/arnakoff/api/issues/215)) ([c824b4e](https://github.com/arnakoff/api/commit/c824b4ebe58650d1528e7a1f9c010310eee8db85)), closes [#209](https://github.com/arnakoff/api/issues/209)

## [0.44.0](https://github.com/arnakoff/api/compare/v0.43.0...v0.44.0) (2026-11-03)


### Features

* **httpapi:** serve new clients when the address limiter is full ([#212](https://github.com/arnakoff/api/issues/212)) ([62040bf](https://github.com/arnakoff/api/commit/62040bfa80ec91ca68b8f301b0fb16939c381528)), closes [#205](https://github.com/arnakoff/api/issues/205)

## [0.43.0](https://github.com/arnakoff/api/compare/v0.42.0...v0.43.0) (2026-11-03)


### Features

* **httpapi:** keep valid api keys in memory for 30 seconds ([#210](https://github.com/arnakoff/api/issues/210)) ([83ce315](https://github.com/arnakoff/api/commit/83ce315e184e77585636f7892b7f964a77debc8c)), closes [#198](https://github.com/arnakoff/api/issues/198)

## [0.42.0](https://github.com/arnakoff/api/compare/v0.41.0...v0.42.0) (2026-11-02)


### Features

* **httpapi:** add a keyless demo check for the site's checker ([#207](https://github.com/arnakoff/api/issues/207)) ([5118973](https://github.com/arnakoff/api/commit/511897306203e509d599f4c3fef04b56bfa0a5d0)), closes [#60](https://github.com/arnakoff/api/issues/60)

## [0.41.0](https://github.com/arnakoff/api/compare/v0.40.0...v0.41.0) (2026-11-02)


### Features

* **httpapi:** limit refused api keys per client address before the key lookup ([#204](https://github.com/arnakoff/api/issues/204)) ([5edea29](https://github.com/arnakoff/api/commit/5edea291a4bc97df0ad39b74d0d8c252f347ba19)), closes [#197](https://github.com/arnakoff/api/issues/197)

## [0.40.0](https://github.com/arnakoff/api/compare/v0.39.0...v0.40.0) (2026-11-02)


### ⚠ BREAKING CHANGES

* **httpapi:** limit requests per api key and per client address ([#201](https://github.com/arnakoff/api/issues/201))

### Features

* **httpapi:** limit requests per api key and per client address ([#201](https://github.com/arnakoff/api/issues/201)) ([2f69a4b](https://github.com/arnakoff/api/commit/2f69a4bf3db58f6c7efdb217c0d983f5ce6e4231)), closes [#58](https://github.com/arnakoff/api/issues/58)

## [0.39.0](https://github.com/arnakoff/api/compare/v0.38.0...v0.39.0) (2026-11-02)


### ⚠ BREAKING CHANGES

* **httpapi:** require an api key on the check routes ([#196](https://github.com/arnakoff/api/issues/196))

### Features

* **httpapi:** require an api key on the check routes ([#196](https://github.com/arnakoff/api/issues/196)) ([c696240](https://github.com/arnakoff/api/commit/c696240e615345a12f42e0a4530b17bbf7e28438)), closes [#57](https://github.com/arnakoff/api/issues/57)

## [0.38.0](https://github.com/arnakoff/api/compare/v0.37.0...v0.38.0) (2026-11-02)


### Features

* **httpapi:** serve get /v1/signals from the signal catalog ([#193](https://github.com/arnakoff/api/issues/193)) ([210225e](https://github.com/arnakoff/api/commit/210225e7531a4096802916a02a3814f1bf7650a4)), closes [#55](https://github.com/arnakoff/api/issues/55)

## [0.37.0](https://github.com/arnakoff/api/compare/v0.36.0...v0.37.0) (2026-11-02)


### Features

* **httpapi:** serve get /v1/sources with the state of each source ([#190](https://github.com/arnakoff/api/issues/190)) ([3395b9b](https://github.com/arnakoff/api/commit/3395b9b664b84be16ad9e938c18b690f8ff904d5)), closes [#54](https://github.com/arnakoff/api/issues/54)

## [0.36.0](https://github.com/arnakoff/api/compare/v0.35.0...v0.36.0) (2026-11-02)


### Features

* **httpapi:** serve post /v1/check/batch with a result per item ([#188](https://github.com/arnakoff/api/issues/188)) ([f32f594](https://github.com/arnakoff/api/commit/f32f594ab26f33734a20bfd946c2bd418a8a2f99)), closes [#53](https://github.com/arnakoff/api/issues/53)

## [0.35.0](https://github.com/arnakoff/api/compare/v0.34.0...v0.35.0) (2026-11-02)


### Features

* **httpapi:** serve get /v1/check with the report of the input ([#185](https://github.com/arnakoff/api/issues/185)) ([76c7996](https://github.com/arnakoff/api/commit/76c7996ebf77b817dbfc4988161cf0a4a29e8fc6)), closes [#52](https://github.com/arnakoff/api/issues/52)

## [0.34.0](https://github.com/arnakoff/api/compare/v0.33.0...v0.34.0) (2026-11-02)


### Features

* **httpapi:** answer every error as an rfc 9457 problem documented on the site ([#183](https://github.com/arnakoff/api/issues/183)) ([7b49288](https://github.com/arnakoff/api/commit/7b4928860418511f52ba602cd3170ade7925bca4)), closes [#51](https://github.com/arnakoff/api/issues/51)

## [0.33.0](https://github.com/arnakoff/api/compare/v0.32.0...v0.33.0) (2026-11-02)


### Features

* **signal:** add medium, beacons and url shorteners to the shared platforms ([#181](https://github.com/arnakoff/api/issues/181)) ([f51650f](https://github.com/arnakoff/api/commit/f51650f62c63b4db2d6923894e3d9025542ddc95)), closes [#172](https://github.com/arnakoff/api/issues/172)

## [0.32.0](https://github.com/arnakoff/api/compare/v0.31.0...v0.32.0) (2026-11-02)


### Features

* **signal:** add bnp paribas to the official domains ([#179](https://github.com/arnakoff/api/issues/179)) ([01039c2](https://github.com/arnakoff/api/commit/01039c2ae356441ceda28de3afc78ac413e019cf)), closes [#166](https://github.com/arnakoff/api/issues/166)

## [0.31.0](https://github.com/arnakoff/api/compare/v0.30.0...v0.31.0) (2026-11-02)


### Features

* **httpapi:** serve the api with bounded timeouts, probes and safe defaults ([#177](https://github.com/arnakoff/api/issues/177)) ([58dbf52](https://github.com/arnakoff/api/commit/58dbf52cf23a82080aa000ecc59717da4f12fd07)), closes [#50](https://github.com/arnakoff/api/issues/50)

## [0.30.0](https://github.com/arnakoff/api/compare/v0.29.1...v0.30.0) (2026-11-02)


### Features

* **httpapi:** add the openapi v1 contract and a contract test harness ([#173](https://github.com/arnakoff/api/issues/173)) ([c9c89d9](https://github.com/arnakoff/api/commit/c9c89d9f07d5083babfdd1be50ab0145406667d2)), closes [#49](https://github.com/arnakoff/api/issues/49)

## [0.29.1](https://github.com/arnakoff/api/compare/v0.29.0...v0.29.1) (2026-11-02)


### Bug Fixes

* **signal:** keep white-list pages on shared platforms from covering them ([#170](https://github.com/arnakoff/api/issues/170)) ([55570cb](https://github.com/arnakoff/api/commit/55570cbf61797e6ae29903b4268c824193125fe0)), closes [#156](https://github.com/arnakoff/api/issues/156)

## [0.29.0](https://github.com/arnakoff/api/compare/v0.28.0...v0.29.0) (2026-11-02)


### Features

* **signal:** flag domains that imitate official sites ([#164](https://github.com/arnakoff/api/issues/164)) ([e5477c8](https://github.com/arnakoff/api/commit/e5477c8c64b90fe1b0d49ecfe3b22f12f38af619)), closes [#47](https://github.com/arnakoff/api/issues/47)

## [0.28.0](https://github.com/arnakoff/api/compare/v0.27.0...v0.28.0) (2026-11-02)


### Features

* **signal:** flag closed companies and bodacc procedures ([#161](https://github.com/arnakoff/api/issues/161)) ([6596aeb](https://github.com/arnakoff/api/commit/6596aeb4dc4bdd1bd2da63d6c6f3c4c1f2ad5fe9)), closes [#44](https://github.com/arnakoff/api/issues/44)
* **signal:** flag unassigned phone numbers and name the range holder ([#163](https://github.com/arnakoff/api/issues/163)) ([fbb6595](https://github.com/arnakoff/api/commit/fbb6595d1380d6ab30c3bb932d19725172171621)), closes [#46](https://github.com/arnakoff/api/issues/46)
* **signal:** state the country holding the account of an iban ([#160](https://github.com/arnakoff/api/issues/160)) ([04f51b9](https://github.com/arnakoff/api/commit/04f51b949c67edfd1c170cd79e6a38a07589934f)), closes [#45](https://github.com/arnakoff/api/issues/45)

## [0.27.0](https://github.com/arnakoff/api/compare/v0.26.0...v0.27.0) (2026-10-30)


### Features

* **signal:** flag domains registered less than 30 days ago ([#158](https://github.com/arnakoff/api/issues/158)) ([dad7b9a](https://github.com/arnakoff/api/commit/dad7b9aa0d0144e33cbce4bb383d25b3e1ec6676)), closes [#43](https://github.com/arnakoff/api/issues/43)

## [0.26.0](https://github.com/arnakoff/api/compare/v0.25.0...v0.26.0) (2026-10-30)


### Features

* **signal:** flag inputs listed on the amf white lists ([#155](https://github.com/arnakoff/api/issues/155)) ([fb1af77](https://github.com/arnakoff/api/commit/fb1af773d899eb6239696470f2430a9dbd5f1ae0)), closes [#42](https://github.com/arnakoff/api/issues/42)

## [0.25.0](https://github.com/arnakoff/api/compare/v0.24.0...v0.25.0) (2026-10-30)


### Features

* **signal:** flag inputs listed on the amf and acpr blacklists ([#153](https://github.com/arnakoff/api/issues/153)) ([fcefbe2](https://github.com/arnakoff/api/commit/fcefbe2521536be97419de866535a89aed510d10)), closes [#41](https://github.com/arnakoff/api/issues/41)

## [0.24.0](https://github.com/arnakoff/api/compare/v0.23.1...v0.24.0) (2026-10-30)


### Features

* **app:** consult the registries concurrently within a global deadline ([#151](https://github.com/arnakoff/api/issues/151)) ([5948d0b](https://github.com/arnakoff/api/commit/5948d0ba24a8dbdf52b57afa2096dfecd390a331)), closes [#40](https://github.com/arnakoff/api/issues/40)

## [0.23.1](https://github.com/arnakoff/api/compare/v0.23.0...v0.23.1) (2026-10-30)


### Bug Fixes

* **registry:** read rdap event dates published without a time ([#147](https://github.com/arnakoff/api/issues/147)) ([f5b29b5](https://github.com/arnakoff/api/commit/f5b29b5f64cf3f05c6d00df98e6cb5ce9d683789)), closes [#144](https://github.com/arnakoff/api/issues/144)

## [0.23.0](https://github.com/arnakoff/api/compare/v0.22.0...v0.23.0) (2026-10-30)


### Features

* **app:** add the evaluation corpus and the eval command ([#145](https://github.com/arnakoff/api/issues/145)) ([ee16d22](https://github.com/arnakoff/api/commit/ee16d2289390f3d114b582f7285c272beb8f09f9)), closes [#39](https://github.com/arnakoff/api/issues/39)

## [0.22.0](https://github.com/arnakoff/api/compare/v0.21.0...v0.22.0) (2026-10-30)


### Features

* **registry:** add the bodacc client for company notices ([#142](https://github.com/arnakoff/api/issues/142)) ([5cd0da4](https://github.com/arnakoff/api/commit/5cd0da49eb8aee9c5153b975ab3ddad4299a49fb)), closes [#37](https://github.com/arnakoff/api/issues/37)

## [0.21.0](https://github.com/arnakoff/api/compare/v0.20.0...v0.21.0) (2026-10-30)


### Features

* **registry:** add the sirene client for the api recherche d'entreprises ([#140](https://github.com/arnakoff/api/issues/140)) ([eba1868](https://github.com/arnakoff/api/commit/eba186844a26f158efe5eb3fcf025d35b48e4290)), closes [#36](https://github.com/arnakoff/api/issues/36)

## [0.20.0](https://github.com/arnakoff/api/compare/v0.19.0...v0.20.0) (2026-10-30)


### Features

* **registry:** add the rdap client with iana bootstrap ([#138](https://github.com/arnakoff/api/issues/138)) ([af32c61](https://github.com/arnakoff/api/commit/af32c61573cebbd75af1fceea751f35caff05015)), closes [#35](https://github.com/arnakoff/api/issues/35)

## [0.19.0](https://github.com/arnakoff/api/compare/v0.18.0...v0.19.0) (2026-10-30)


### Features

* **sync:** schedule source syncs, alert on failure and hot-reload snapshots ([#136](https://github.com/arnakoff/api/issues/136)) ([e728ff2](https://github.com/arnakoff/api/commit/e728ff25e7fc5215f467e7fb8e2347dc963a185a)), closes [#33](https://github.com/arnakoff/api/issues/33)

## [0.18.0](https://github.com/arnakoff/api/compare/v0.17.0...v0.18.0) (2026-10-30)


### Features

* **source:** add the acpr blacklist connector ([#134](https://github.com/arnakoff/api/issues/134)) ([126df50](https://github.com/arnakoff/api/commit/126df50bdc00ddf2291720f87ecc0d022dd94179)), closes [#31](https://github.com/arnakoff/api/issues/31)

## [0.17.0](https://github.com/arnakoff/api/compare/v0.16.0...v0.17.0) (2026-10-29)


### Features

* **source:** add the arcep numbering plan connector ([#132](https://github.com/arnakoff/api/issues/132)) ([77be82e](https://github.com/arnakoff/api/commit/77be82e42dbde8075b871c497e1b01f5021415da)), closes [#32](https://github.com/arnakoff/api/issues/32)

## [0.16.0](https://github.com/arnakoff/api/compare/v0.15.0...v0.16.0) (2026-10-29)


### Features

* **source:** add the amf psan and biens divers white lists ([#130](https://github.com/arnakoff/api/issues/130)) ([8ae444c](https://github.com/arnakoff/api/commit/8ae444c617e4a4b4a778f93fd69aacb6182b3bae))

## [0.15.0](https://github.com/arnakoff/api/compare/v0.14.0...v0.15.0) (2026-10-29)


### Features

* **source:** add the amf blacklist connector ([#128](https://github.com/arnakoff/api/issues/128)) ([dfaa19e](https://github.com/arnakoff/api/commit/dfaa19e6b2fb2ff401e845aa982d1b7b13961868)), closes [#28](https://github.com/arnakoff/api/issues/28)

## [0.14.0](https://github.com/arnakoff/api/compare/v0.13.0...v0.14.0) (2026-10-29)


### Features

* **sync:** store snapshots on disk with a checksum and atomic replace ([#126](https://github.com/arnakoff/api/issues/126)) ([fdfc6c5](https://github.com/arnakoff/api/commit/fdfc6c5c93fd20fea3723b295648f9cffe009fe8)), closes [#27](https://github.com/arnakoff/api/issues/27)

## [0.13.0](https://github.com/arnakoff/api/compare/v0.12.0...v0.13.0) (2026-10-29)


### Features

* **app:** add the sync use case with validation and atomic swap ([#124](https://github.com/arnakoff/api/issues/124)) ([1f48990](https://github.com/arnakoff/api/commit/1f489905ffb58fa5d27ca700dd1593ed84c51b97)), closes [#26](https://github.com/arnakoff/api/issues/26)

## [0.12.0](https://github.com/arnakoff/api/compare/v0.11.0...v0.12.0) (2026-10-29)


### Features

* **platform:** wire the redacting logger into every command ([#122](https://github.com/arnakoff/api/issues/122)) ([cbed709](https://github.com/arnakoff/api/commit/cbed7098a752bdfbb1df6db2761d260530d6810f)), closes [#117](https://github.com/arnakoff/api/issues/117)

## [0.11.0](https://github.com/arnakoff/api/compare/v0.10.0...v0.11.0) (2026-10-29)


### Features

* **platform:** add the bounded in-memory cache with call deduplication ([#120](https://github.com/arnakoff/api/issues/120)) ([835aca8](https://github.com/arnakoff/api/commit/835aca8b3f1463d38bc856a10ee5777912569aea)), closes [#24](https://github.com/arnakoff/api/issues/24)

## [0.10.0](https://github.com/arnakoff/api/compare/v0.9.0...v0.10.0) (2026-10-29)


### Features

* **platform:** add the typed configuration from environment variables ([#118](https://github.com/arnakoff/api/issues/118)) ([8974e2c](https://github.com/arnakoff/api/commit/8974e2c498539af8cf5c17dae909997212f8153e)), closes [#23](https://github.com/arnakoff/api/issues/23)

## [0.9.0](https://github.com/arnakoff/api/compare/v0.8.0...v0.9.0) (2026-10-29)


### Features

* **platform:** add the redacting structured logger ([#115](https://github.com/arnakoff/api/issues/115)) ([119b2ce](https://github.com/arnakoff/api/commit/119b2ce0c210fea3d0e3641d905aa3bdd281ef0d)), closes [#22](https://github.com/arnakoff/api/issues/22)

## [0.8.0](https://github.com/arnakoff/api/compare/v0.7.0...v0.8.0) (2026-10-29)


### Features

* **platform:** add the outbound HTTP client ([#113](https://github.com/arnakoff/api/issues/113)) ([8ff1b03](https://github.com/arnakoff/api/commit/8ff1b0333d6a7bf64ac5913076bab6f5542489cf)), closes [#21](https://github.com/arnakoff/api/issues/21)

## [0.7.0](https://github.com/arnakoff/api/compare/v0.6.0...v0.7.0) (2026-10-29)


### Features

* **domain:** detect the kind of a pasted input ([#111](https://github.com/arnakoff/api/issues/111)) ([6df76cf](https://github.com/arnakoff/api/commit/6df76cf6e2ab849de65c902f3f5845d9b8164446)), closes [#19](https://github.com/arnakoff/api/issues/19)

## [0.6.0](https://github.com/arnakoff/api/compare/v0.5.0...v0.6.0) (2026-10-29)


### Features

* **domain:** add the SIREN and SIRET value objects ([#109](https://github.com/arnakoff/api/issues/109)) ([33eddf5](https://github.com/arnakoff/api/commit/33eddf5fd7cf600128a2f5f567172ccb6f4b478f)), closes [#18](https://github.com/arnakoff/api/issues/18)

## [0.5.0](https://github.com/arnakoff/api/compare/v0.4.0...v0.5.0) (2026-10-29)


### Features

* **domain:** add the IBAN value object ([#107](https://github.com/arnakoff/api/issues/107)) ([666e7f7](https://github.com/arnakoff/api/commit/666e7f7d03b4138a9e5dd6b1b1cacf7f30a11758)), closes [#17](https://github.com/arnakoff/api/issues/17)

## [0.4.0](https://github.com/arnakoff/api/compare/v0.3.0...v0.4.0) (2026-10-29)


### Features

* **domain:** add the phone value object ([#104](https://github.com/arnakoff/api/issues/104)) ([19c4ea2](https://github.com/arnakoff/api/commit/19c4ea2058a0e2fd0e85cafff3fce6671d7afca1)), closes [#16](https://github.com/arnakoff/api/issues/16)

## [0.3.0](https://github.com/arnakoff/api/compare/v0.2.0...v0.3.0) (2026-10-29)


### Features

* **domain:** add the email value object ([#102](https://github.com/arnakoff/api/issues/102)) ([7bb9a4e](https://github.com/arnakoff/api/commit/7bb9a4e2ee272947be65f0e39485006f62a5c396)), closes [#15](https://github.com/arnakoff/api/issues/15)

## [0.2.0](https://github.com/arnakoff/api/compare/v0.1.0...v0.2.0) (2026-10-29)


### Features

* **domain:** add the domain value object ([#98](https://github.com/arnakoff/api/issues/98)) ([9902fe7](https://github.com/arnakoff/api/commit/9902fe76e17888cac4ab96efcc4d81129763d89a)), closes [#14](https://github.com/arnakoff/api/issues/14)

## 0.1.0 (2026-10-29)


### Features

* **domain:** add signal, level, source and report types ([#96](https://github.com/arnakoff/api/issues/96)) ([5997506](https://github.com/arnakoff/api/commit/59975066650844ffb96a3e204f81a7c21e4a42fa)), closes [#13](https://github.com/arnakoff/api/issues/13)
