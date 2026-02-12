# Changelog

## [0.1.42](https://github.com/hansjlachmann/openerp/compare/v0.1.41...v0.1.42) (2026-02-12)


### Bug Fixes

* send codeunit errors as error events instead of success completion ([60d18a7](https://github.com/hansjlachmann/openerp/commit/60d18a7a22ceb60b22e0e0cad0e78d3d518c21bf))

## [0.1.41](https://github.com/hansjlachmann/openerp/compare/v0.1.40...v0.1.41) (2026-02-12)


### Bug Fixes

* show NAV errors as toast instead of briefly-visible modal ([dae05b0](https://github.com/hansjlachmann/openerp/commit/dae05b0a7966c32b1961b87042317e9fbe9bdca8))

## [0.1.40](https://github.com/hansjlachmann/openerp/compare/v0.1.39...v0.1.40) (2026-02-12)


### Features

* add parameter field to Job Queue for generic report runner ([f2b7034](https://github.com/hansjlachmann/openerp/commit/f2b7034f538f349bec6ea45550012053df8d5c5e))


### Bug Fixes

* don't stop polling when startjob POST times out on heavy reports ([62aae27](https://github.com/hansjlachmann/openerp/commit/62aae2787e3538a2a71bc03048edbc36b8a44a45))
* remove ineffectual assignments flagged by golangci-lint ([86ac26c](https://github.com/hansjlachmann/openerp/commit/86ac26ce30db824d928a70f07b5ad62d598f5dfd))
* stop polling when startjob fails and no progress is ever seen ([6426dad](https://github.com/hansjlachmann/openerp/commit/6426dadfad4915773c5c39afaced294af1c9042b))

## [0.1.39](https://github.com/hansjlachmann/openerp/compare/v0.1.38...v0.1.39) (2026-02-12)


### Bug Fixes

* show 0% progress when report job starts ([159bbd3](https://github.com/hansjlachmann/openerp/commit/159bbd3e254f9e182cb572a5919ef9bed2204100))

## [0.1.38](https://github.com/hansjlachmann/openerp/compare/v0.1.37...v0.1.38) (2026-02-12)


### Bug Fixes

* extract PDF path from StartJob response, not CheckJob ([76c02a1](https://github.com/hansjlachmann/openerp/commit/76c02a1d57dc62fb540a654df2f9c96ae8487377))

## [0.1.37](https://github.com/hansjlachmann/openerp/compare/v0.1.36...v0.1.37) (2026-02-12)


### Bug Fixes

* correct PDF endpoint URL and add PdfPath parameter ([51ae017](https://github.com/hansjlachmann/openerp/commit/51ae017386e72dc0113844b97f6e285d86314099))

## [0.1.36](https://github.com/hansjlachmann/openerp/compare/v0.1.35...v0.1.36) (2026-02-12)


### Bug Fixes

* add initial delay before checkjob poll and fix PDF download endpoint ([bd92f7c](https://github.com/hansjlachmann/openerp/commit/bd92f7c987b38ff078c816867cf7adcf8d427dc7))
* update CheckJob call to POST with CompanyName parameter ([f120093](https://github.com/hansjlachmann/openerp/commit/f1200937316c1c9ec4194c3c982da59edb786a45))

## [0.1.35](https://github.com/hansjlachmann/openerp/compare/v0.1.34...v0.1.35) (2026-02-11)


### Features

* move all frontend translations to backend YAML files ([d441ee7](https://github.com/hansjlachmann/openerp/commit/d441ee7b0a4594846b5b746d5cbafc226d317061))
* render boolean fields as checkboxes on card pages ([8f30c55](https://github.com/hansjlachmann/openerp/commit/8f30c55963af15c7e63d0a6e4779716fec6d4fa7))


### Bug Fixes

* correct Norwegian menu item translations ([533fab5](https://github.com/hansjlachmann/openerp/commit/533fab59c88434aa243a9bbe195a8e79e5970f26))
* update E2E test to accept translation keys when backend is unavailable ([c6d3eb9](https://github.com/hansjlachmann/openerp/commit/c6d3eb9ad148312cc72d60345738400a3f13f244))
* use full page reload after login to apply user's language ([688fbbd](https://github.com/hansjlachmann/openerp/commit/688fbbd0521510d189ed22244371ee13ddde5c70))

## [0.1.34](https://github.com/hansjlachmann/openerp/compare/v0.1.33...v0.1.34) (2026-02-11)


### Features

* add JOBQUEUE menu profile and fix nb-NO translation gaps ([c2c01e6](https://github.com/hansjlachmann/openerp/commit/c2c01e653c58f94294c6da9c56ed94ea3d82a10d))
* extract CreateJobQueueEntry helper, add menu i18n, and fix missing translations ([5f3a327](https://github.com/hansjlachmann/openerp/commit/5f3a327377d75c40d7ff86a771cb8707db263b77))

## [0.1.33](https://github.com/hansjlachmann/openerp/compare/v0.1.32...v0.1.33) (2026-02-10)


### Bug Fixes

* default empty string to first option for Option fields (NAV/BC behavior) ([9e0b75e](https://github.com/hansjlachmann/openerp/commit/9e0b75e7a5ac0fb1c4989ab270a4d92e999f8a0b))

## [0.1.32](https://github.com/hansjlachmann/openerp/compare/v0.1.31...v0.1.32) (2026-02-10)


### Features

* send company name in NavReportRunner startjob request ([1614a61](https://github.com/hansjlachmann/openerp/commit/1614a614432cd4267d0a8bce1ca7d0ce0dbbe030))

## [0.1.31](https://github.com/hansjlachmann/openerp/compare/v0.1.30...v0.1.31) (2026-02-10)


### Bug Fixes

* update .env version and prevent CI race condition in .env commit step ([7a64f62](https://github.com/hansjlachmann/openerp/commit/7a64f62958feadae9e018683520a2bb6ce55e636))

## [0.1.30](https://github.com/hansjlachmann/openerp/compare/v0.1.29...v0.1.30) (2026-02-10)


### Features

* add APP_VERSION support to docker-compose ([aa0833f](https://github.com/hansjlachmann/openerp/commit/aa0833f89875cc6a136045fdbf279c84065cbe47))
* add automatic table relation validation in tablegen ([ce4d816](https://github.com/hansjlachmann/openerp/commit/ce4d816c42dcfed28e85358872b4d1d07b914721))
* add cancel button to progress modal for long-running jobs ([de5f16a](https://github.com/hansjlachmann/openerp/commit/de5f16a069f224504871d95996ae6a1f7dc573a6))
* add codeunit helper functions Message() and Error() ([bf963e1](https://github.com/hansjlachmann/openerp/commit/bf963e18636dc1353f9b9a8e2c433ee9b74185a0))
* add codeunit to generate random customer ledger entries ([bed58b8](https://github.com/hansjlachmann/openerp/commit/bed58b8dae35f99712f76c90188c0f99fb52736e))
* add company switcher and codeunit dialog support ([a3cf6e0](https://github.com/hansjlachmann/openerp/commit/a3cf6e09232c9d93499a657e19b90bbea5b05ec6))
* add Confirm() helper function for codeunits ([cf7ebb2](https://github.com/hansjlachmann/openerp/commit/cf7ebb26eae0f2e1be21ed673feb32c348b32b13))
* add dark mode support to login page and layout ([4b96f50](https://github.com/hansjlachmann/openerp/commit/4b96f50d80f67760c75231c4ef1231f2f5f800c9))
* add detailed logging to NavReportRunner for debugging ([ae15a6f](https://github.com/hansjlachmann/openerp/commit/ae15a6fec6344e3e663189b33ce3d70f1a856eb0))
* add Escape key navigation in list pages (NAV/BC behavior) ([2e71d20](https://github.com/hansjlachmann/openerp/commit/2e71d200bbfb0ba73abd9a288af395d82c5265e8))
* add extension support with extmerge tool ([5022860](https://github.com/hansjlachmann/openerp/commit/5022860c146ad0c653650a40c19db84628b80eb4))
* add F8 to copy value from cell above (NAV/BC behavior) ([422ded6](https://github.com/hansjlachmann/openerp/commit/422ded6494de75c06b3336c0c9ea2ab56b2a1915))
* add focus_field property for Card pages ([496fe6b](https://github.com/hansjlachmann/openerp/commit/496fe6b5028e79e4e8a547a092bf8b710ede3246))
* add i18n for messages and display company name in menu bar ([5c6afdc](https://github.com/hansjlachmann/openerp/commit/5c6afdc762a569e57a671f5ff866b93ac5f693f0))
* add i18n for messages and display company name in menu bar ([d163ee8](https://github.com/hansjlachmann/openerp/commit/d163ee87fb59ed7fad34a5cc09f82b042129961a))
* add Job Queue Entry table and list page ([c3698b7](https://github.com/hansjlachmann/openerp/commit/c3698b75a503039f4be94144493f0cab59b7c816))
* add Job Queue table with Run action and code optimizations ([71a4ed5](https://github.com/hansjlachmann/openerp/commit/71a4ed5ca0172955a994fb1a2fd161465bbcfbd6))
* add keyboard shortcuts for List page actions ([529392a](https://github.com/hansjlachmann/openerp/commit/529392aaca67f4fac4531fc93ebac49c84fbae53))
* add Language table with relation to User ([82829d1](https://github.com/hansjlachmann/openerp/commit/82829d1ff2225340cf1ac3109ae253c6fd37a542))
* add logging for POST request to NAV service ([f0f82d3](https://github.com/hansjlachmann/openerp/commit/f0f82d3801ebb40e2c4ae7a00a0f703c702b0975))
* Add logout functionality and enforce authentication ([157c4c7](https://github.com/hansjlachmann/openerp/commit/157c4c7b1170e6a9aef271d0495ff939e0f47400))
* add multi-column lookup dropdown with type-ahead search ([40e2c64](https://github.com/hansjlachmann/openerp/commit/40e2c648377e94d0c76a023468bf67498a0d8b19))
* add multi-language support and breadcrumb navigation ([d4eaf9c](https://github.com/hansjlachmann/openerp/commit/d4eaf9c7726b36b8692a21c18741429866dbf584))
* add NAV-style progress dialog for codeunits ([bf2c859](https://github.com/hansjlachmann/openerp/commit/bf2c8593675b850a2dcc5f5c297cefe94f9c2ee4))
* add NavReportRunner codeunit for external report generation ([2f68dba](https://github.com/hansjlachmann/openerp/commit/2f68dba280c7f0119b1750266630e4b9eaad43fc))
* add Option field support and improve modal UX ([9aa35dc](https://github.com/hansjlachmann/openerp/commit/9aa35dce6ee08c8ebda572d88ddc3e49bc9795b2))
* add permission enforcement middleware for table API routes ([8f1a455](https://github.com/hansjlachmann/openerp/commit/8f1a455c7309248cfc4ae1e518f1add5cb67ebe7))
* add Permission table and session-based RBAC ([85d8c89](https://github.com/hansjlachmann/openerp/commit/85d8c8921652ffd422018cac785c14b98aabb718))
* add production docker-compose with pre-built images ([39a36cf](https://github.com/hansjlachmann/openerp/commit/39a36cfbb5fb9def3a0d6323e536d09bc701110b))
* add session helper functions to codeunits package ([f94ecc0](https://github.com/hansjlachmann/openerp/commit/f94ecc0be3b901c97e9b4453d53c73ff131ebff4))
* add table relation validation with field revert on error ([e2a9a52](https://github.com/hansjlachmann/openerp/commit/e2a9a520ea2029eaafe8a53f605706d65e19fd5f))
* add translation_key field and fix list page empty row handling ([60a236d](https://github.com/hansjlachmann/openerp/commit/60a236d78628e0737ceb9ce95da38ca3accc5c14))
* add translations for permission tables and seed default roles ([a4e93b0](https://github.com/hansjlachmann/openerp/commit/a4e93b05064d1eaee48e445ada4fea494de04d4a))
* add UI components and improve editable list functionality ([55332e0](https://github.com/hansjlachmann/openerp/commit/55332e01b528c358b587efbac56a605ad3de04b9))
* Add user authentication and management system ([3748247](https://github.com/hansjlachmann/openerp/commit/37482471eeedc5746a953af37c89c682a29bfe48))
* Add user preferences system and BC-style filter support ([5176453](https://github.com/hansjlachmann/openerp/commit/517645361b5f10fa5f857148c697b4f2102ce038))
* add User Role and User Member permission tables ([40b0cc3](https://github.com/hansjlachmann/openerp/commit/40b0cc3f5e528a562e129b0e6257f6e27c479642))
* add User Role card page and Permission list page ([86ef6f0](https://github.com/hansjlachmann/openerp/commit/86ef6f0212f40cf1e25269e435a709dc755bdac1))
* Add user-specific customizations and fix phone number field ([922e72e](https://github.com/hansjlachmann/openerp/commit/922e72e7398551489230a388836acd918041a8f7))
* add versioned database migration system ([c9b0837](https://github.com/hansjlachmann/openerp/commit/c9b0837ef007a7f01d1d2caf85bf34fbae0226ea))
* auto-update .env with version on release ([efb3462](https://github.com/hansjlachmann/openerp/commit/efb3462fc6f107cb7d4d33601bdc47b4d791e20a))
* block editing when new record save fails (e.g., duplicate) ([6f7efb5](https://github.com/hansjlachmann/openerp/commit/6f7efb5ed4638e54f1de5d23615f149d08168e36))
* codeunits self-declare progress support via UsesProgress() ([c0dff60](https://github.com/hansjlachmann/openerp/commit/c0dff606a2fc3713a2ecd52142cc1760b7d7b6dc))
* display version in menu bar ([6b0efad](https://github.com/hansjlachmann/openerp/commit/6b0efaddab92f208956ce5f2c4b71856b34b70a5))
* Docker containerization with PostgreSQL and UI improvements ([03191d2](https://github.com/hansjlachmann/openerp/commit/03191d212c1d2fdec2a8c5bd5aaed909364c8582))
* Docker containerization with PostgreSQL and UI improvements ([0ec60f0](https://github.com/hansjlachmann/openerp/commit/0ec60f064fd6399ceae04148472d06b298b8e8f1))
* enforce company access control, composite PK support, and global table sync ([1d4d8d1](https://github.com/hansjlachmann/openerp/commit/1d4d8d1ee0798197465562f6f9495315b2faae29))
* fire-and-forget POST to NAV service, poll immediately ([f78bc4c](https://github.com/hansjlachmann/openerp/commit/f78bc4c2b1db0f380e317373c5467f652304ea21))
* Front-end UI ([1f12fb7](https://github.com/hansjlachmann/openerp/commit/1f12fb7705c2f0510b67071459133a9810d761c3))
* generate 20-char alphanumeric JobId with timestamp ([7158b91](https://github.com/hansjlachmann/openerp/commit/7158b9154f3189699f92a0628f6adc3d45684342))
* genereric error messages from foundation layer ([8c2e393](https://github.com/hansjlachmann/openerp/commit/8c2e393349daa351c44d20880addce857c51fd0e))
* implement generic codeunit registry pattern ([a1dc552](https://github.com/hansjlachmann/openerp/commit/a1dc552dd7b77dff87efe36e9154d2a958e713cf))
* implement user-assignable menu system ([9a29e55](https://github.com/hansjlachmann/openerp/commit/9a29e55f6c0845ad11c2055bfd0f4279cfe15a32))
* improve Customer Card modal UX and Edit button functionality ([8d2faa5](https://github.com/hansjlachmann/openerp/commit/8d2faa51519008fd7d45c59c096260131d43a300))
* improve Customer Card UX and fix button states ([8c6eff0](https://github.com/hansjlachmann/openerp/commit/8c6eff0c4502ff313a1c9566cc1b8a64a858201a))
* improve list page edit mode keyboard navigation (NAV/BC behavior) ([a1293e4](https://github.com/hansjlachmann/openerp/commit/a1293e4540215f7ee35c5fa714e53ce6a28ac7eb))
* improve report polling - check PDF endpoint every 5 seconds ([c963f3f](https://github.com/hansjlachmann/openerp/commit/c963f3fbba8edb3faba1dc827e0d75c8025343bd))
* make menu groups data-driven from YAML ([6c26ce4](https://github.com/hansjlachmann/openerp/commit/6c26ce4d4ebbdc50f9186610f8fc9097799ddb9d))
* Redesign FilterPane with Business Central-style Views ([e7895b2](https://github.com/hansjlachmann/openerp/commit/e7895b2caeead15f70b163208ee7c6ebb5c47d80))
* Reorganize page header buttons layout ([b83ec40](https://github.com/hansjlachmann/openerp/commit/b83ec40215523b20f4e1d86cafe8cf2f6123096c))
* show lookup dropdowns in non-edit mode on card pages ([96baa2e](https://github.com/hansjlachmann/openerp/commit/96baa2e29aadef8a986f0ca2524dbea5af27a1a4))
* smart auto-save with change detection and UI improvements ([e56250c](https://github.com/hansjlachmann/openerp/commit/e56250c3d9bcba6c804c59e140751a4fcd41f0db))
* UI components, multi-language support, and editable list improvements ([#17](https://github.com/hansjlachmann/openerp/issues/17)) ([e8faa99](https://github.com/hansjlachmann/openerp/commit/e8faa993df6c01cc39f47917c70c38baf3f45d7e))
* update NavReportRunner to use dedicated PDF endpoint ([a525860](https://github.com/hansjlachmann/openerp/commit/a525860d1bb55908b5b91d8605679aa5bf2c22e6))
* use LookupDropdown in list page edit cells, fix composite PK SQL ([30f3ed4](https://github.com/hansjlachmann/openerp/commit/30f3ed4cc150e4b64aede3fccd9f137004df12b9))


### Bug Fixes

* add Back to List action to User Card ([e9478f0](https://github.com/hansjlachmann/openerp/commit/e9478f0cd314415a98bb404d0836e59772c770b6))
* add empty ID validation to ModifyRecord and DeleteRecord ([319594e](https://github.com/hansjlachmann/openerp/commit/319594ecec5515419dab6098bf52fbea2f186800))
* add empty ID validation to ModifyRecord and DeleteRecord ([059eb1f](https://github.com/hansjlachmann/openerp/commit/059eb1f9976931cd3e69ada8b02a97e7f9ba92be))
* Add ensureTableExists to create tables on-demand from metadata ([71b3e4e](https://github.com/hansjlachmann/openerp/commit/71b3e4ebc9c958c9334e3f33fb0adcfd0e76c5ec))
* Add missing caption for Payment Terms 'active' field ([bcdd16f](https://github.com/hansjlachmann/openerp/commit/bcdd16f1f7b167980db1745c738d3227490e2838))
* Add missing fyne.io/fyne/v2 import for GUI ([ea8e5f9](https://github.com/hansjlachmann/openerp/commit/ea8e5f9e367127e075ef7db4489df01fea1a716d))
* add table_relation to language field in User card page ([d5bee50](https://github.com/hansjlachmann/openerp/commit/d5bee5049d8eca40ebefaa15bbef592437693f5b))
* allow CORS from all origins in production ([81bac7a](https://github.com/hansjlachmann/openerp/commit/81bac7a8e6d9ddadec475b35a6fe98aa4d4233ec))
* card page keyboard shortcuts and dark mode default ([ac738a0](https://github.com/hansjlachmann/openerp/commit/ac738a034c22d4e2ef79e3cb72d145930be114ba))
* consistent row height between edit and view mode in list pages ([0aa0278](https://github.com/hansjlachmann/openerp/commit/0aa02785859fdecbc10c052d9ab8562cb80e53df))
* correct Norwegian translation for preferences ([71e234d](https://github.com/hansjlachmann/openerp/commit/71e234dc98ab53da68938612a200e41baca6d0a1))
* correct TypeScript type for lookup data in getTableOptionsAndLookups ([6f09987](https://github.com/hansjlachmann/openerp/commit/6f099872b744b89410c9346854560fdfe366bdac))
* Create FieldDefinition table in OpenDatabase for backward compatibility ([58f7d6a](https://github.com/hansjlachmann/openerp/commit/58f7d6a424a257c627ce08205e6b11b2815d6d79))
* CreateTable now inserts marker record so table shows in ListTables ([a77a732](https://github.com/hansjlachmann/openerp/commit/a77a73241501f4c50d0994aadd89b2557dfd98a6))
* day and night mode ([069c52f](https://github.com/hansjlachmann/openerp/commit/069c52f5fac2a77aeac7924116af9906e20aa67b))
* delayed insert for composite PKs with optional fields ([5cbbc08](https://github.com/hansjlachmann/openerp/commit/5cbbc08db3a03df73724e7caa0e9f616a24584ec))
* focus first input when opening card page modal ([2bfdc48](https://github.com/hansjlachmann/openerp/commit/2bfdc48673cd77cab40125f6144102588d46525e))
* handle NULL database values and improve new record detection ([7657b8f](https://github.com/hansjlachmann/openerp/commit/7657b8fcbb92d84bfe38dfefcd8ec0bf75387895))
* hardcode report ID to 121 for NAV proxy service ([40b6d4c](https://github.com/hansjlachmann/openerp/commit/40b6d4c0201bacc40bcd3f9d150c1dcf86f6021e))
* improve progress polling - poll immediately and robust parsing ([853a9e9](https://github.com/hansjlachmann/openerp/commit/853a9e909178c029977bbc3390b7b2ff37b64052))
* improve version text visibility in menu bar ([71cdc8a](https://github.com/hansjlachmann/openerp/commit/71cdc8a04cc93122cb763fc9b5addb9ee9e473bf))
* list page border now ends at last record ([3008003](https://github.com/hansjlachmann/openerp/commit/30080037762ab03b30276c8bf4a4a3da6e138b93))
* make list page column headers sticky when scrolling ([72cdcd8](https://github.com/hansjlachmann/openerp/commit/72cdcd8e011c728afb586d18280fb5a29bff6dca))
* make list page column headers sticky when scrolling ([304ac9c](https://github.com/hansjlachmann/openerp/commit/304ac9c019bf6c3e2a347b28eab44f6867e70566))
* make list page column headers sticky when scrolling ([ceca5da](https://github.com/hansjlachmann/openerp/commit/ceca5dac8259bd9eee6ffcc376bed567c06180c0))
* modal card close functionality and keyboard shortcut handling ([482b471](https://github.com/hansjlachmann/openerp/commit/482b471cdd8d6cb862568c6f39f1f6039fe06bae))
* modal card close functionality and keyboard shortcut handling ([61e3181](https://github.com/hansjlachmann/openerp/commit/61e31816eb5511dcd2864be11baa4ae2d76efd5e))
* Move authentication check to layout load function ([c0c4ae7](https://github.com/hansjlachmann/openerp/commit/c0c4ae7f39165810f4278ccd857e6fc8676a5883))
* pass database type to codeunit for PostgreSQL compatibility ([85fd8e5](https://github.com/hansjlachmann/openerp/commit/85fd8e57168024f199aa046507f632a6673162eb))
* prevent creating records with empty primary key ([83837f8](https://github.com/hansjlachmann/openerp/commit/83837f89443c8dcd882a1013d72b35b18bee1407))
* prevent creating records with empty primary key ([2a78917](https://github.com/hansjlachmann/openerp/commit/2a78917355f9a01532337c15d62496d6933bf01d))
* prevent duplicate new rows and allow typing in lookup fields ([48c74c1](https://github.com/hansjlachmann/openerp/commit/48c74c1886eb192619ee597e70e8ba7f5e154757))
* readme ([380234a](https://github.com/hansjlachmann/openerp/commit/380234a1612bbebd214d1f6f3a420216f909fc06))
* readme ([a9d5f5e](https://github.com/hansjlachmann/openerp/commit/a9d5f5e8a2a79f4392f9397758bdd4079b2603e6))
* reduce progress polling interval to 1 second ([df033b4](https://github.com/hansjlachmann/openerp/commit/df033b49b9fce8c70603840e45fca99ab5440d90))
* reduce row height for option dropdowns in list page ([312070a](https://github.com/hansjlachmann/openerp/commit/312070a7d0448d8c55284956fd6a6155e59fbb9e))
* register Language table in table registry ([2ed3a63](https://github.com/hansjlachmann/openerp/commit/2ed3a63ba4b7d38e6a1d58802ff2d2d87fc67660))
* Remove nested go.mod files and update import paths ([e981e79](https://github.com/hansjlachmann/openerp/commit/e981e7939e8f93609058ce4e0039eccdfe18ff66))
* remove unused confirmChan field from Dialog struct ([4aa14fc](https://github.com/hansjlachmann/openerp/commit/4aa14fc09b6bd6a791f2a7c635a31cafabd3aecb))
* remove unused escapeJSON function ([d351735](https://github.com/hansjlachmann/openerp/commit/d35173518a9286964ec3516d732c69e832131b66))
* resolve go vet errors in backend code ([04c2711](https://github.com/hansjlachmann/openerp/commit/04c271180f1fd98b3e81f9fc0208fe7262a93f4d))
* resolve golangci-lint errors ([676c217](https://github.com/hansjlachmann/openerp/commit/676c21785321b0f0573d47977914601511e0774a))
* resolve remaining a11y warnings ([5609e66](https://github.com/hansjlachmann/openerp/commit/5609e668b5c9fc675b62d63cfd7a6940443fadd5))
* resolve Svelte 5 props_invalid_value error for lookup dropdowns ([e2b9955](https://github.com/hansjlachmann/openerp/commit/e2b99556d2ac6da951f19d68e32284283362cfd1))
* resolve Svelte 5 warnings ([3dee4cc](https://github.com/hansjlachmann/openerp/commit/3dee4cc5f559b20f2a01e795ca43afe00ee0a608))
* resolve svelte-check warnings ([b17b2c6](https://github.com/hansjlachmann/openerp/commit/b17b2c6ade2701221979e625f69cf935bcb63a62))
* resolve TypeScript errors in frontend build ([3a80767](https://github.com/hansjlachmann/openerp/commit/3a8076746000bebba5df53ca37b1005648dc9f66))
* resolve unreachable code errors in generated table files ([61eb6a1](https://github.com/hansjlachmann/openerp/commit/61eb6a1c39d97c21d981e2fedd360c6ee48760e1))
* support composite primary keys in get/modify/delete API endpoints ([9a80a40](https://github.com/hansjlachmann/openerp/commit/9a80a4022a80e34e67bb1baeb66ca0b77b41a15b))
* support HTTP (non-secure) contexts for toast notifications ([3c55da2](https://github.com/hansjlachmann/openerp/commit/3c55da21f722d94c7f18b20f15f426864ada6b1d))
* update gitignore ([f099415](https://github.com/hansjlachmann/openerp/commit/f099415cebdf80077df3520f51339fa87a045445))
* update report dialog ([76a84a9](https://github.com/hansjlachmann/openerp/commit/76a84a903518c0b6e6b9f4029e78124b1ede870a))
* use crypto/rand for job IDs and propagate POST errors ([bd6dc5c](https://github.com/hansjlachmann/openerp/commit/bd6dc5c0ccdcf3368b361001e333a1e1bb7e602e))
* use generic action button captions ([8ed4ca3](https://github.com/hansjlachmann/openerp/commit/8ed4ca36d4fbb290c961c83f89f2c3d6dbf41b13))
* use getRecordId helper for delete and row click in PageRenderer ([14fd744](https://github.com/hansjlachmann/openerp/commit/14fd744d17a8f6919afdf0493077328124518d4a))
* use per-goroutine session context for codeunit execution ([aec5208](https://github.com/hansjlachmann/openerp/commit/aec52080c09d27fc818631a05400bb2051f06cb1))
* wait for CheckJob 100% before fetching PDF ([2a48cd8](https://github.com/hansjlachmann/openerp/commit/2a48cd8de72c5ab601ca8bbbe51396190c04e3ff))


### Code Refactoring

* Auto-initialize tables and remove Object Designer ([87b9a97](https://github.com/hansjlachmann/openerp/commit/87b9a97dc52ff3839237c4578a69ec4b6e3ac729))
* consolidate confirmation modal to shared store ([a9e2df3](https://github.com/hansjlachmann/openerp/commit/a9e2df359405862e13c23e3f40407b62240bff51))
* consolidate duplicate code across frontend ([d90a1c6](https://github.com/hansjlachmann/openerp/commit/d90a1c64bb34a62f44fd070e22dbfcaef1a27261))
* consolidate duplicate code and improve code organization ([d9ee3af](https://github.com/hansjlachmann/openerp/commit/d9ee3afe75ae1ec55650a2126c5e988511aee435))
* create centralized localStorage utility ([7169a9e](https://github.com/hansjlachmann/openerp/commit/7169a9e1e28461c3c3574de0e263d035d791cc88))
* Extract duplicate code into reusable utilities and components ([6320d32](https://github.com/hansjlachmann/openerp/commit/6320d3202ec347d3088eb226cfbf798ec3663f65))
* extract shared utilities for record handling and API helpers ([632ce42](https://github.com/hansjlachmann/openerp/commit/632ce425055f4f2e5516bf214cc495f984b82ab0))
* extract visibility logic and remove dead code ([7a611a6](https://github.com/hansjlachmann/openerp/commit/7a611a6b0da214d4b006ffd185ff1445a4b7a400))
* implement generic Table interface for API handlers ([c525b7b](https://github.com/hansjlachmann/openerp/commit/c525b7bdc73c1b9cce95545caa7da2a445af7d63))
* make frontend fully generic using primary_key from page definitions ([b7ac3e7](https://github.com/hansjlachmann/openerp/commit/b7ac3e7b9c94d4a54132d6c046e238e9d1798402))
* Move pages folder to business logic layer ([dd201cc](https://github.com/hansjlachmann/openerp/commit/dd201cc066b15e17ce6c182f6a985095341220e1))
* remove legacy hardcoded customer code ([7d52c62](https://github.com/hansjlachmann/openerp/commit/7d52c628c658943dbbc845cf1fd99409e88d35b5))
* separate generated table code from manual business logic ([d10462f](https://github.com/hansjlachmann/openerp/commit/d10462fb1dec3f7731c90adb6ea2469335a5387b))
* translate all hardcoded API error messages ([9135d22](https://github.com/hansjlachmann/openerp/commit/9135d227c692309dc90c5d3d97084c8fa82ddddc))
* translate all hardcoded API error messages ([7cac3f9](https://github.com/hansjlachmann/openerp/commit/7cac3f9763be0806e975edfc4fa69cc9064acefe))


### Documentation

* add CLAUDE.md with project rules and conventions ([7ebba1c](https://github.com/hansjlachmann/openerp/commit/7ebba1cbbc5f6d136dcdfa21d6944459ce683274))
* add migration system documentation ([0184056](https://github.com/hansjlachmann/openerp/commit/0184056aa0379744bd7a694ea6bb182e08b50a51))
* add required runtimes Go 1.24 and Node.js 22 to CLAUDE.md ([1bf564f](https://github.com/hansjlachmann/openerp/commit/1bf564fcd680bfd6778141c4d97f4c2f1315c29f))
* add Svelte 5 runes and i18n rules to CLAUDE.md ([8e82fc4](https://github.com/hansjlachmann/openerp/commit/8e82fc4cb67ee9f125c01fd5abc2e78e11c57b76))


### CI/CD

* add code coverage with Codecov and frontend tests ([8c0a6f5](https://github.com/hansjlachmann/openerp/commit/8c0a6f51d67d597ef5209f892a6a45cdc1140639))
* add Docker build and push to GitHub Container Registry ([d4ce779](https://github.com/hansjlachmann/openerp/commit/d4ce7798221b5e649d45ea9e5f15292c565e8f0b))
* add GitHub Actions build and lint workflow ([ddc10b4](https://github.com/hansjlachmann/openerp/commit/ddc10b4d970a7db50b28abe1e669b91242a8fe3a))
* add Go test step with race detection and coverage ([644d876](https://github.com/hansjlachmann/openerp/commit/644d876e6d2685b8d882f308dc4148c5af7fa61f))
* add golangci-lint step ([62ada5c](https://github.com/hansjlachmann/openerp/commit/62ada5c358000ba534db8bbb703cdc842508a4d6))
* add multi-arch Docker builds (AMD64 + ARM64) ([50091e6](https://github.com/hansjlachmann/openerp/commit/50091e622b65d8629f86ffd4c4f456b9f2ef7d7c))
* add Playwright E2E testing ([e7ebae5](https://github.com/hansjlachmann/openerp/commit/e7ebae5fee99f943d20f31bc6a3b4a6e6d6662e2))
* add release-please for automated releases ([58c99bc](https://github.com/hansjlachmann/openerp/commit/58c99bc41cc874490018eaf8d65b913325fbaf5f))
* added "needs" to release-please ([c1d89d3](https://github.com/hansjlachmann/openerp/commit/c1d89d383d6206949404b8a5cb2df8e5e8ff6d95))
* merge build and release workflows into single file ([5998956](https://github.com/hansjlachmann/openerp/commit/599895664ceea43e10269bc1b04bea3b45a6662f))
* only build Docker images on release ([8961ef1](https://github.com/hansjlachmann/openerp/commit/8961ef108a920a9fcd8c2954e7b280148fd41799))

## [0.1.29](https://github.com/hansjlachmann/openerp/compare/v0.1.28...v0.1.29) (2026-02-07)


### Features

* add permission enforcement middleware for table API routes ([8f1a455](https://github.com/hansjlachmann/openerp/commit/8f1a455c7309248cfc4ae1e518f1add5cb67ebe7))
* add Permission table and session-based RBAC ([85d8c89](https://github.com/hansjlachmann/openerp/commit/85d8c8921652ffd422018cac785c14b98aabb718))
* add translations for permission tables and seed default roles ([a4e93b0](https://github.com/hansjlachmann/openerp/commit/a4e93b05064d1eaee48e445ada4fea494de04d4a))
* add User Role and User Member permission tables ([40b0cc3](https://github.com/hansjlachmann/openerp/commit/40b0cc3f5e528a562e129b0e6257f6e27c479642))
* add User Role card page and Permission list page ([86ef6f0](https://github.com/hansjlachmann/openerp/commit/86ef6f0212f40cf1e25269e435a709dc755bdac1))


### Documentation

* add CLAUDE.md with project rules and conventions ([7ebba1c](https://github.com/hansjlachmann/openerp/commit/7ebba1cbbc5f6d136dcdfa21d6944459ce683274))
* add required runtimes Go 1.24 and Node.js 22 to CLAUDE.md ([1bf564f](https://github.com/hansjlachmann/openerp/commit/1bf564fcd680bfd6778141c4d97f4c2f1315c29f))
* add Svelte 5 runes and i18n rules to CLAUDE.md ([8e82fc4](https://github.com/hansjlachmann/openerp/commit/8e82fc4cb67ee9f125c01fd5abc2e78e11c57b76))

## [0.1.28](https://github.com/hansjlachmann/openerp/compare/v0.1.27...v0.1.28) (2026-02-06)


### Bug Fixes

* readme ([380234a](https://github.com/hansjlachmann/openerp/commit/380234a1612bbebd214d1f6f3a420216f909fc06))
* readme ([a9d5f5e](https://github.com/hansjlachmann/openerp/commit/a9d5f5e8a2a79f4392f9397758bdd4079b2603e6))

## [0.1.27](https://github.com/hansjlachmann/openerp/compare/v0.1.26...v0.1.27) (2026-02-02)


### Bug Fixes

* use crypto/rand for job IDs and propagate POST errors ([bd6dc5c](https://github.com/hansjlachmann/openerp/commit/bd6dc5c0ccdcf3368b361001e333a1e1bb7e602e))
* wait for CheckJob 100% before fetching PDF ([2a48cd8](https://github.com/hansjlachmann/openerp/commit/2a48cd8de72c5ab601ca8bbbe51396190c04e3ff))

## [0.1.26](https://github.com/hansjlachmann/openerp/compare/v0.1.25...v0.1.26) (2026-01-29)


### Features

* add logging for POST request to NAV service ([f0f82d3](https://github.com/hansjlachmann/openerp/commit/f0f82d3801ebb40e2c4ae7a00a0f703c702b0975))
* fire-and-forget POST to NAV service, poll immediately ([f78bc4c](https://github.com/hansjlachmann/openerp/commit/f78bc4c2b1db0f380e317373c5467f652304ea21))
* improve report polling - check PDF endpoint every 5 seconds ([c963f3f](https://github.com/hansjlachmann/openerp/commit/c963f3fbba8edb3faba1dc827e0d75c8025343bd))


### Bug Fixes

* hardcode report ID to 121 for NAV proxy service ([40b6d4c](https://github.com/hansjlachmann/openerp/commit/40b6d4c0201bacc40bcd3f9d150c1dcf86f6021e))
* reduce progress polling interval to 1 second ([df033b4](https://github.com/hansjlachmann/openerp/commit/df033b49b9fce8c70603840e45fca99ab5440d90))

## [0.1.25](https://github.com/hansjlachmann/openerp/compare/v0.1.24...v0.1.25) (2026-01-29)


### Features

* add detailed logging to NavReportRunner for debugging ([ae15a6f](https://github.com/hansjlachmann/openerp/commit/ae15a6fec6344e3e663189b33ce3d70f1a856eb0))

## [0.1.24](https://github.com/hansjlachmann/openerp/compare/v0.1.23...v0.1.24) (2026-01-28)


### Features

* add cancel button to progress modal for long-running jobs ([de5f16a](https://github.com/hansjlachmann/openerp/commit/de5f16a069f224504871d95996ae6a1f7dc573a6))


### Bug Fixes

* improve progress polling - poll immediately and robust parsing ([853a9e9](https://github.com/hansjlachmann/openerp/commit/853a9e909178c029977bbc3390b7b2ff37b64052))

## [0.1.23](https://github.com/hansjlachmann/openerp/compare/v0.1.22...v0.1.23) (2026-01-28)


### Features

* add NavReportRunner codeunit for external report generation ([2f68dba](https://github.com/hansjlachmann/openerp/commit/2f68dba280c7f0119b1750266630e4b9eaad43fc))
* generate 20-char alphanumeric JobId with timestamp ([7158b91](https://github.com/hansjlachmann/openerp/commit/7158b9154f3189699f92a0628f6adc3d45684342))
* update NavReportRunner to use dedicated PDF endpoint ([a525860](https://github.com/hansjlachmann/openerp/commit/a525860d1bb55908b5b91d8605679aa5bf2c22e6))


### Bug Fixes

* remove unused escapeJSON function ([d351735](https://github.com/hansjlachmann/openerp/commit/d35173518a9286964ec3516d732c69e832131b66))

## [0.1.22](https://github.com/hansjlachmann/openerp/compare/v0.1.21...v0.1.22) (2026-01-28)


### Features

* add Escape key navigation in list pages (NAV/BC behavior) ([2e71d20](https://github.com/hansjlachmann/openerp/commit/2e71d200bbfb0ba73abd9a288af395d82c5265e8))
* add F8 to copy value from cell above (NAV/BC behavior) ([422ded6](https://github.com/hansjlachmann/openerp/commit/422ded6494de75c06b3336c0c9ea2ab56b2a1915))
* improve list page edit mode keyboard navigation (NAV/BC behavior) ([a1293e4](https://github.com/hansjlachmann/openerp/commit/a1293e4540215f7ee35c5fa714e53ce6a28ac7eb))


### Bug Fixes

* consistent row height between edit and view mode in list pages ([0aa0278](https://github.com/hansjlachmann/openerp/commit/0aa02785859fdecbc10c052d9ab8562cb80e53df))

## [0.1.21](https://github.com/hansjlachmann/openerp/compare/v0.1.20...v0.1.21) (2026-01-28)


### Features

* add Confirm() helper function for codeunits ([cf7ebb2](https://github.com/hansjlachmann/openerp/commit/cf7ebb26eae0f2e1be21ed673feb32c348b32b13))


### Bug Fixes

* remove unused confirmChan field from Dialog struct ([4aa14fc](https://github.com/hansjlachmann/openerp/commit/4aa14fc09b6bd6a791f2a7c635a31cafabd3aecb))

## [0.1.20](https://github.com/hansjlachmann/openerp/compare/v0.1.19...v0.1.20) (2026-01-28)


### Bug Fixes

* support HTTP (non-secure) contexts for toast notifications ([3c55da2](https://github.com/hansjlachmann/openerp/commit/3c55da21f722d94c7f18b20f15f426864ada6b1d))

## [0.1.19](https://github.com/hansjlachmann/openerp/compare/v0.1.18...v0.1.19) (2026-01-28)


### Features

* add production docker-compose with pre-built images ([39a36cf](https://github.com/hansjlachmann/openerp/commit/39a36cfbb5fb9def3a0d6323e536d09bc701110b))

## [0.1.18](https://github.com/hansjlachmann/openerp/compare/v0.1.17...v0.1.18) (2026-01-28)


### Features

* add APP_VERSION support to docker-compose ([aa0833f](https://github.com/hansjlachmann/openerp/commit/aa0833f89875cc6a136045fdbf279c84065cbe47))
* add NAV-style progress dialog for codeunits ([bf2c859](https://github.com/hansjlachmann/openerp/commit/bf2c8593675b850a2dcc5f5c297cefe94f9c2ee4))
* auto-update .env with version on release ([efb3462](https://github.com/hansjlachmann/openerp/commit/efb3462fc6f107cb7d4d33601bdc47b4d791e20a))
* codeunits self-declare progress support via UsesProgress() ([c0dff60](https://github.com/hansjlachmann/openerp/commit/c0dff606a2fc3713a2ecd52142cc1760b7d7b6dc))


### Bug Fixes

* allow CORS from all origins in production ([81bac7a](https://github.com/hansjlachmann/openerp/commit/81bac7a8e6d9ddadec475b35a6fe98aa4d4233ec))
* use per-goroutine session context for codeunit execution ([aec5208](https://github.com/hansjlachmann/openerp/commit/aec52080c09d27fc818631a05400bb2051f06cb1))

## [0.1.17](https://github.com/hansjlachmann/openerp/compare/v0.1.16...v0.1.17) (2026-01-27)


### Documentation

* add migration system documentation ([0184056](https://github.com/hansjlachmann/openerp/commit/0184056aa0379744bd7a694ea6bb182e08b50a51))

## [0.1.16](https://github.com/hansjlachmann/openerp/compare/v0.1.15...v0.1.16) (2026-01-27)


### Features

* add session helper functions to codeunits package ([f94ecc0](https://github.com/hansjlachmann/openerp/commit/f94ecc0be3b901c97e9b4453d53c73ff131ebff4))

## [0.1.15](https://github.com/hansjlachmann/openerp/compare/v0.1.14...v0.1.15) (2026-01-27)


### Features

* add codeunit helper functions Message() and Error() ([bf963e1](https://github.com/hansjlachmann/openerp/commit/bf963e18636dc1353f9b9a8e2c433ee9b74185a0))

## [0.1.14](https://github.com/hansjlachmann/openerp/compare/v0.1.13...v0.1.14) (2026-01-27)


### Features

* add company switcher and codeunit dialog support ([a3cf6e0](https://github.com/hansjlachmann/openerp/commit/a3cf6e09232c9d93499a657e19b90bbea5b05ec6))

## [0.1.13](https://github.com/hansjlachmann/openerp/compare/v0.1.12...v0.1.13) (2026-01-27)


### Features

* add Job Queue Entry table and list page ([c3698b7](https://github.com/hansjlachmann/openerp/commit/c3698b75a503039f4be94144493f0cab59b7c816))

## [0.1.12](https://github.com/hansjlachmann/openerp/compare/v0.1.11...v0.1.12) (2026-01-27)


### Features

* add Job Queue table with Run action and code optimizations ([71a4ed5](https://github.com/hansjlachmann/openerp/commit/71a4ed5ca0172955a994fb1a2fd161465bbcfbd6))


### Bug Fixes

* resolve unreachable code errors in generated table files ([61eb6a1](https://github.com/hansjlachmann/openerp/commit/61eb6a1c39d97c21d981e2fedd360c6ee48760e1))

## [0.1.11](https://github.com/hansjlachmann/openerp/compare/v0.1.10...v0.1.11) (2026-01-27)


### Features

* add extension support with extmerge tool ([5022860](https://github.com/hansjlachmann/openerp/commit/5022860c146ad0c653650a40c19db84628b80eb4))

## [0.1.10](https://github.com/hansjlachmann/openerp/compare/v0.1.9...v0.1.10) (2026-01-27)


### CI/CD

* only build Docker images on release ([8961ef1](https://github.com/hansjlachmann/openerp/commit/8961ef108a920a9fcd8c2954e7b280148fd41799))

## [0.1.9](https://github.com/hansjlachmann/openerp/compare/v0.1.8...v0.1.9) (2026-01-27)


### Bug Fixes

* add Back to List action to User Card ([e9478f0](https://github.com/hansjlachmann/openerp/commit/e9478f0cd314415a98bb404d0836e59772c770b6))
* card page keyboard shortcuts and dark mode default ([ac738a0](https://github.com/hansjlachmann/openerp/commit/ac738a034c22d4e2ef79e3cb72d145930be114ba))

## [0.1.8](https://github.com/hansjlachmann/openerp/compare/v0.1.7...v0.1.8) (2026-01-27)


### Features

* display version in menu bar ([6b0efad](https://github.com/hansjlachmann/openerp/commit/6b0efaddab92f208956ce5f2c4b71856b34b70a5))


### Bug Fixes

* improve version text visibility in menu bar ([71cdc8a](https://github.com/hansjlachmann/openerp/commit/71cdc8a04cc93122cb763fc9b5addb9ee9e473bf))

## [0.1.7](https://github.com/hansjlachmann/openerp/compare/v0.1.6...v0.1.7) (2026-01-27)


### Features

* add versioned database migration system ([c9b0837](https://github.com/hansjlachmann/openerp/commit/c9b0837ef007a7f01d1d2caf85bf34fbae0226ea))

## [0.1.6](https://github.com/hansjlachmann/openerp/compare/v0.1.5...v0.1.6) (2026-01-25)


### Features

* add codeunit to generate random customer ledger entries ([bed58b8](https://github.com/hansjlachmann/openerp/commit/bed58b8dae35f99712f76c90188c0f99fb52736e))
* implement generic codeunit registry pattern ([a1dc552](https://github.com/hansjlachmann/openerp/commit/a1dc552dd7b77dff87efe36e9154d2a958e713cf))


### Bug Fixes

* pass database type to codeunit for PostgreSQL compatibility ([85fd8e5](https://github.com/hansjlachmann/openerp/commit/85fd8e57168024f199aa046507f632a6673162eb))

## [0.1.5](https://github.com/hansjlachmann/openerp/compare/v0.1.4...v0.1.5) (2026-01-25)


### Features

* make menu groups data-driven from YAML ([6c26ce4](https://github.com/hansjlachmann/openerp/commit/6c26ce4d4ebbdc50f9186610f8fc9097799ddb9d))

## [0.1.4](https://github.com/hansjlachmann/openerp/compare/v0.1.3...v0.1.4) (2026-01-24)


### CI/CD

* add multi-arch Docker builds (AMD64 + ARM64) ([50091e6](https://github.com/hansjlachmann/openerp/commit/50091e622b65d8629f86ffd4c4f456b9f2ef7d7c))

## [0.1.3](https://github.com/hansjlachmann/openerp/compare/v0.1.2...v0.1.3) (2026-01-23)


### CI/CD

* add Playwright E2E testing ([e7ebae5](https://github.com/hansjlachmann/openerp/commit/e7ebae5fee99f943d20f31bc6a3b4a6e6d6662e2))

## [0.1.2](https://github.com/hansjlachmann/openerp/compare/v0.1.1...v0.1.2) (2026-01-23)


### CI/CD

* add code coverage with Codecov and frontend tests ([8c0a6f5](https://github.com/hansjlachmann/openerp/commit/8c0a6f51d67d597ef5209f892a6a45cdc1140639))
* added "needs" to release-please ([c1d89d3](https://github.com/hansjlachmann/openerp/commit/c1d89d383d6206949404b8a5cb2df8e5e8ff6d95))

## [0.1.1](https://github.com/hansjlachmann/openerp/compare/v0.1.0...v0.1.1) (2026-01-23)


### Features

* add automatic table relation validation in tablegen ([ce4d816](https://github.com/hansjlachmann/openerp/commit/ce4d816c42dcfed28e85358872b4d1d07b914721))
* add dark mode support to login page and layout ([4b96f50](https://github.com/hansjlachmann/openerp/commit/4b96f50d80f67760c75231c4ef1231f2f5f800c9))
* add focus_field property for Card pages ([496fe6b](https://github.com/hansjlachmann/openerp/commit/496fe6b5028e79e4e8a547a092bf8b710ede3246))
* add i18n for messages and display company name in menu bar ([5c6afdc](https://github.com/hansjlachmann/openerp/commit/5c6afdc762a569e57a671f5ff866b93ac5f693f0))
* add i18n for messages and display company name in menu bar ([d163ee8](https://github.com/hansjlachmann/openerp/commit/d163ee87fb59ed7fad34a5cc09f82b042129961a))
* add keyboard shortcuts for List page actions ([529392a](https://github.com/hansjlachmann/openerp/commit/529392aaca67f4fac4531fc93ebac49c84fbae53))
* add Language table with relation to User ([82829d1](https://github.com/hansjlachmann/openerp/commit/82829d1ff2225340cf1ac3109ae253c6fd37a542))
* Add logout functionality and enforce authentication ([157c4c7](https://github.com/hansjlachmann/openerp/commit/157c4c7b1170e6a9aef271d0495ff939e0f47400))
* add multi-column lookup dropdown with type-ahead search ([40e2c64](https://github.com/hansjlachmann/openerp/commit/40e2c648377e94d0c76a023468bf67498a0d8b19))
* add multi-language support and breadcrumb navigation ([d4eaf9c](https://github.com/hansjlachmann/openerp/commit/d4eaf9c7726b36b8692a21c18741429866dbf584))
* add Option field support and improve modal UX ([9aa35dc](https://github.com/hansjlachmann/openerp/commit/9aa35dce6ee08c8ebda572d88ddc3e49bc9795b2))
* add table relation validation with field revert on error ([e2a9a52](https://github.com/hansjlachmann/openerp/commit/e2a9a520ea2029eaafe8a53f605706d65e19fd5f))
* add translation_key field and fix list page empty row handling ([60a236d](https://github.com/hansjlachmann/openerp/commit/60a236d78628e0737ceb9ce95da38ca3accc5c14))
* add UI components and improve editable list functionality ([55332e0](https://github.com/hansjlachmann/openerp/commit/55332e01b528c358b587efbac56a605ad3de04b9))
* Add user authentication and management system ([3748247](https://github.com/hansjlachmann/openerp/commit/37482471eeedc5746a953af37c89c682a29bfe48))
* Add user preferences system and BC-style filter support ([5176453](https://github.com/hansjlachmann/openerp/commit/517645361b5f10fa5f857148c697b4f2102ce038))
* Add user-specific customizations and fix phone number field ([922e72e](https://github.com/hansjlachmann/openerp/commit/922e72e7398551489230a388836acd918041a8f7))
* block editing when new record save fails (e.g., duplicate) ([6f7efb5](https://github.com/hansjlachmann/openerp/commit/6f7efb5ed4638e54f1de5d23615f149d08168e36))
* Docker containerization with PostgreSQL and UI improvements ([03191d2](https://github.com/hansjlachmann/openerp/commit/03191d212c1d2fdec2a8c5bd5aaed909364c8582))
* Docker containerization with PostgreSQL and UI improvements ([0ec60f0](https://github.com/hansjlachmann/openerp/commit/0ec60f064fd6399ceae04148472d06b298b8e8f1))
* Front-end UI ([1f12fb7](https://github.com/hansjlachmann/openerp/commit/1f12fb7705c2f0510b67071459133a9810d761c3))
* genereric error messages from foundation layer ([8c2e393](https://github.com/hansjlachmann/openerp/commit/8c2e393349daa351c44d20880addce857c51fd0e))
* implement user-assignable menu system ([9a29e55](https://github.com/hansjlachmann/openerp/commit/9a29e55f6c0845ad11c2055bfd0f4279cfe15a32))
* improve Customer Card modal UX and Edit button functionality ([8d2faa5](https://github.com/hansjlachmann/openerp/commit/8d2faa51519008fd7d45c59c096260131d43a300))
* improve Customer Card UX and fix button states ([8c6eff0](https://github.com/hansjlachmann/openerp/commit/8c6eff0c4502ff313a1c9566cc1b8a64a858201a))
* Redesign FilterPane with Business Central-style Views ([e7895b2](https://github.com/hansjlachmann/openerp/commit/e7895b2caeead15f70b163208ee7c6ebb5c47d80))
* Reorganize page header buttons layout ([b83ec40](https://github.com/hansjlachmann/openerp/commit/b83ec40215523b20f4e1d86cafe8cf2f6123096c))
* show lookup dropdowns in non-edit mode on card pages ([96baa2e](https://github.com/hansjlachmann/openerp/commit/96baa2e29aadef8a986f0ca2524dbea5af27a1a4))
* smart auto-save with change detection and UI improvements ([e56250c](https://github.com/hansjlachmann/openerp/commit/e56250c3d9bcba6c804c59e140751a4fcd41f0db))
* UI components, multi-language support, and editable list improvements ([#17](https://github.com/hansjlachmann/openerp/issues/17)) ([e8faa99](https://github.com/hansjlachmann/openerp/commit/e8faa993df6c01cc39f47917c70c38baf3f45d7e))


### Bug Fixes

* add empty ID validation to ModifyRecord and DeleteRecord ([319594e](https://github.com/hansjlachmann/openerp/commit/319594ecec5515419dab6098bf52fbea2f186800))
* add empty ID validation to ModifyRecord and DeleteRecord ([059eb1f](https://github.com/hansjlachmann/openerp/commit/059eb1f9976931cd3e69ada8b02a97e7f9ba92be))
* Add ensureTableExists to create tables on-demand from metadata ([71b3e4e](https://github.com/hansjlachmann/openerp/commit/71b3e4ebc9c958c9334e3f33fb0adcfd0e76c5ec))
* Add missing caption for Payment Terms 'active' field ([bcdd16f](https://github.com/hansjlachmann/openerp/commit/bcdd16f1f7b167980db1745c738d3227490e2838))
* Add missing fyne.io/fyne/v2 import for GUI ([ea8e5f9](https://github.com/hansjlachmann/openerp/commit/ea8e5f9e367127e075ef7db4489df01fea1a716d))
* add table_relation to language field in User card page ([d5bee50](https://github.com/hansjlachmann/openerp/commit/d5bee5049d8eca40ebefaa15bbef592437693f5b))
* correct Norwegian translation for preferences ([71e234d](https://github.com/hansjlachmann/openerp/commit/71e234dc98ab53da68938612a200e41baca6d0a1))
* correct TypeScript type for lookup data in getTableOptionsAndLookups ([6f09987](https://github.com/hansjlachmann/openerp/commit/6f099872b744b89410c9346854560fdfe366bdac))
* Create FieldDefinition table in OpenDatabase for backward compatibility ([58f7d6a](https://github.com/hansjlachmann/openerp/commit/58f7d6a424a257c627ce08205e6b11b2815d6d79))
* CreateTable now inserts marker record so table shows in ListTables ([a77a732](https://github.com/hansjlachmann/openerp/commit/a77a73241501f4c50d0994aadd89b2557dfd98a6))
* day and night mode ([069c52f](https://github.com/hansjlachmann/openerp/commit/069c52f5fac2a77aeac7924116af9906e20aa67b))
* focus first input when opening card page modal ([2bfdc48](https://github.com/hansjlachmann/openerp/commit/2bfdc48673cd77cab40125f6144102588d46525e))
* handle NULL database values and improve new record detection ([7657b8f](https://github.com/hansjlachmann/openerp/commit/7657b8fcbb92d84bfe38dfefcd8ec0bf75387895))
* list page border now ends at last record ([3008003](https://github.com/hansjlachmann/openerp/commit/30080037762ab03b30276c8bf4a4a3da6e138b93))
* make list page column headers sticky when scrolling ([72cdcd8](https://github.com/hansjlachmann/openerp/commit/72cdcd8e011c728afb586d18280fb5a29bff6dca))
* make list page column headers sticky when scrolling ([304ac9c](https://github.com/hansjlachmann/openerp/commit/304ac9c019bf6c3e2a347b28eab44f6867e70566))
* make list page column headers sticky when scrolling ([ceca5da](https://github.com/hansjlachmann/openerp/commit/ceca5dac8259bd9eee6ffcc376bed567c06180c0))
* modal card close functionality and keyboard shortcut handling ([482b471](https://github.com/hansjlachmann/openerp/commit/482b471cdd8d6cb862568c6f39f1f6039fe06bae))
* modal card close functionality and keyboard shortcut handling ([61e3181](https://github.com/hansjlachmann/openerp/commit/61e31816eb5511dcd2864be11baa4ae2d76efd5e))
* Move authentication check to layout load function ([c0c4ae7](https://github.com/hansjlachmann/openerp/commit/c0c4ae7f39165810f4278ccd857e6fc8676a5883))
* prevent creating records with empty primary key ([83837f8](https://github.com/hansjlachmann/openerp/commit/83837f89443c8dcd882a1013d72b35b18bee1407))
* prevent creating records with empty primary key ([2a78917](https://github.com/hansjlachmann/openerp/commit/2a78917355f9a01532337c15d62496d6933bf01d))
* reduce row height for option dropdowns in list page ([312070a](https://github.com/hansjlachmann/openerp/commit/312070a7d0448d8c55284956fd6a6155e59fbb9e))
* register Language table in table registry ([2ed3a63](https://github.com/hansjlachmann/openerp/commit/2ed3a63ba4b7d38e6a1d58802ff2d2d87fc67660))
* Remove nested go.mod files and update import paths ([e981e79](https://github.com/hansjlachmann/openerp/commit/e981e7939e8f93609058ce4e0039eccdfe18ff66))
* resolve go vet errors in backend code ([04c2711](https://github.com/hansjlachmann/openerp/commit/04c271180f1fd98b3e81f9fc0208fe7262a93f4d))
* resolve golangci-lint errors ([676c217](https://github.com/hansjlachmann/openerp/commit/676c21785321b0f0573d47977914601511e0774a))
* resolve remaining a11y warnings ([5609e66](https://github.com/hansjlachmann/openerp/commit/5609e668b5c9fc675b62d63cfd7a6940443fadd5))
* resolve Svelte 5 props_invalid_value error for lookup dropdowns ([e2b9955](https://github.com/hansjlachmann/openerp/commit/e2b99556d2ac6da951f19d68e32284283362cfd1))
* resolve Svelte 5 warnings ([3dee4cc](https://github.com/hansjlachmann/openerp/commit/3dee4cc5f559b20f2a01e795ca43afe00ee0a608))
* resolve svelte-check warnings ([b17b2c6](https://github.com/hansjlachmann/openerp/commit/b17b2c6ade2701221979e625f69cf935bcb63a62))
* resolve TypeScript errors in frontend build ([3a80767](https://github.com/hansjlachmann/openerp/commit/3a8076746000bebba5df53ca37b1005648dc9f66))
* update gitignore ([f099415](https://github.com/hansjlachmann/openerp/commit/f099415cebdf80077df3520f51339fa87a045445))
* use generic action button captions ([8ed4ca3](https://github.com/hansjlachmann/openerp/commit/8ed4ca36d4fbb290c961c83f89f2c3d6dbf41b13))
* use getRecordId helper for delete and row click in PageRenderer ([14fd744](https://github.com/hansjlachmann/openerp/commit/14fd744d17a8f6919afdf0493077328124518d4a))


### Code Refactoring

* Auto-initialize tables and remove Object Designer ([87b9a97](https://github.com/hansjlachmann/openerp/commit/87b9a97dc52ff3839237c4578a69ec4b6e3ac729))
* consolidate confirmation modal to shared store ([a9e2df3](https://github.com/hansjlachmann/openerp/commit/a9e2df359405862e13c23e3f40407b62240bff51))
* consolidate duplicate code across frontend ([d90a1c6](https://github.com/hansjlachmann/openerp/commit/d90a1c64bb34a62f44fd070e22dbfcaef1a27261))
* consolidate duplicate code and improve code organization ([d9ee3af](https://github.com/hansjlachmann/openerp/commit/d9ee3afe75ae1ec55650a2126c5e988511aee435))
* create centralized localStorage utility ([7169a9e](https://github.com/hansjlachmann/openerp/commit/7169a9e1e28461c3c3574de0e263d035d791cc88))
* Extract duplicate code into reusable utilities and components ([6320d32](https://github.com/hansjlachmann/openerp/commit/6320d3202ec347d3088eb226cfbf798ec3663f65))
* extract shared utilities for record handling and API helpers ([632ce42](https://github.com/hansjlachmann/openerp/commit/632ce425055f4f2e5516bf214cc495f984b82ab0))
* extract visibility logic and remove dead code ([7a611a6](https://github.com/hansjlachmann/openerp/commit/7a611a6b0da214d4b006ffd185ff1445a4b7a400))
* implement generic Table interface for API handlers ([c525b7b](https://github.com/hansjlachmann/openerp/commit/c525b7bdc73c1b9cce95545caa7da2a445af7d63))
* make frontend fully generic using primary_key from page definitions ([b7ac3e7](https://github.com/hansjlachmann/openerp/commit/b7ac3e7b9c94d4a54132d6c046e238e9d1798402))
* Move pages folder to business logic layer ([dd201cc](https://github.com/hansjlachmann/openerp/commit/dd201cc066b15e17ce6c182f6a985095341220e1))
* remove legacy hardcoded customer code ([7d52c62](https://github.com/hansjlachmann/openerp/commit/7d52c628c658943dbbc845cf1fd99409e88d35b5))
* separate generated table code from manual business logic ([d10462f](https://github.com/hansjlachmann/openerp/commit/d10462fb1dec3f7731c90adb6ea2469335a5387b))
* translate all hardcoded API error messages ([9135d22](https://github.com/hansjlachmann/openerp/commit/9135d227c692309dc90c5d3d97084c8fa82ddddc))
* translate all hardcoded API error messages ([7cac3f9](https://github.com/hansjlachmann/openerp/commit/7cac3f9763be0806e975edfc4fa69cc9064acefe))


### CI/CD

* add Docker build and push to GitHub Container Registry ([d4ce779](https://github.com/hansjlachmann/openerp/commit/d4ce7798221b5e649d45ea9e5f15292c565e8f0b))
* add GitHub Actions build and lint workflow ([ddc10b4](https://github.com/hansjlachmann/openerp/commit/ddc10b4d970a7db50b28abe1e669b91242a8fe3a))
* add Go test step with race detection and coverage ([644d876](https://github.com/hansjlachmann/openerp/commit/644d876e6d2685b8d882f308dc4148c5af7fa61f))
* add golangci-lint step ([62ada5c](https://github.com/hansjlachmann/openerp/commit/62ada5c358000ba534db8bbb703cdc842508a4d6))
* add release-please for automated releases ([58c99bc](https://github.com/hansjlachmann/openerp/commit/58c99bc41cc874490018eaf8d65b913325fbaf5f))
* merge build and release workflows into single file ([5998956](https://github.com/hansjlachmann/openerp/commit/599895664ceea43e10269bc1b04bea3b45a6662f))
