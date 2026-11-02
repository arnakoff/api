# Changelog

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
