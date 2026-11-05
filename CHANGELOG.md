# Changelog

## Change Types

- **Features**: New features or improvements
- **Bug Fixes**: Bug fixes and patches
- **Breaking Changes**: Incompatible changes

## Commit Guidelines

All notable changes to this project will be documented in this file. See [Conventional Commits](https://www.conventionalcommits.org/) specification when making commits.

---
### [v0.1.0](https://github.com/sichang824/skm/commits/v0.1.0) (2026-11-05)

### Commits

* fix(ci): close test databases and separate frontend release builds ([3f57732](https://github.com/sichang824/skm/commit/3f5773219a1a9ad09851a61feae52dc104c32da9))
* fix(ci): install a reproducible Windows UCRT compiler ([ca56918](https://github.com/sichang824/skm/commit/ca56918d2dac36dbee94aa22f6eb13bffd4d92dd))
* fix(release): verify universal architectures and align macOS bundle version ([3c1ff61](https://github.com/sichang824/skm/commit/3c1ff6119886ac9dd716ba9c4399f10b4d921dad))
* feat(release): build macOS and Windows releases with RustyTag versions ([d4ef7ed](https://github.com/sichang824/skm/commit/d4ef7ed373c60790f2d01fd1763a30b2a5a3e9ab))
* refactor(search): improve skill query precision and documentation ([dfc7d56](https://github.com/sichang824/skm/commit/dfc7d56a5cd310532d054fde4eac9c6e6fbac58f))
* feat(digest): implement compact skill digest output for LLM context injection ([5d681b0](https://github.com/sichang824/skm/commit/5d681b05df10911f6a44ca0c77c1bdbb4fd2f387))
* feat(exec): skill package.json 可执行清单与 exec 执行引擎（第 1–8 期） ([7bcc227](https://github.com/sichang824/skm/commit/7bcc227d32abd486ab64d6c3a52e33d08ad53557))
* feat(config): enhance environment loading and testing ([5416eb9](https://github.com/sichang824/skm/commit/5416eb94926c12eeb8f03210cd43bd72aee883ba))
* fix(catalog): cascade provider delete and widen attach dialog ([df0591f](https://github.com/sichang824/skm/commit/df0591f21af22ed3d012b8bb9c375737d6db5f2c))
* feat: add sync-copies functionality for skill management ([d6c4b0b](https://github.com/sichang824/skm/commit/d6c4b0bd377e14129414351d807387e33f7a575f))
* feat: enhance skill relation management and error handling ([e5a187c](https://github.com/sichang824/skm/commit/e5a187cd8728a968fa36c48087a329b49c199429))
* feat: enhance skill management and desktop integration ([ff4c490](https://github.com/sichang824/skm/commit/ff4c4906cf3b0fca29bdf766618054df5bbf8f95))
* feat: update .gitignore to include .DS_Store and add .to configuration file with SKILL.md documentation ([5ee7fe4](https://github.com/sichang824/skm/commit/5ee7fe417c64e65ca13fec5c4e80842c3532c785))
* feat: add desktop CLI management and configuration ([fb50b5e](https://github.com/sichang824/skm/commit/fb50b5e29e7e07dc1669828036b08d1de6bf6767))
* feat: update default SQLite database path to ~/.skm/app.db and add tests for configuration ([59dc2e8](https://github.com/sichang824/skm/commit/59dc2e830ec801f8ee36c46f1361dd35b685894a))
* feat: add interface preview section with dashboard and skills catalog images to README ([364cf1c](https://github.com/sichang824/skm/commit/364cf1c827c6e603503d6bb69fdb348291e4642e))
* feat: add contributing guidelines, license, and localized README for SKM ([05a8bb2](https://github.com/sichang824/skm/commit/05a8bb2be36e82518a26350fb272553f2210cf2b))
* feat: update Wails CLI installation in Makefile and enhance wails.json configuration ([5803300](https://github.com/sichang824/skm/commit/5803300f46f455b54de2726600711e13096f8bf6))
* feat: add provider icons and enhance provider display in UI ([8575a09](https://github.com/sichang824/skm/commit/8575a09fadea13dd9885eea95b1e60dc8a05b3fd))
* feat: enhance desktop experience with draggable regions and improved macOS title bar handling ([47d2f50](https://github.com/sichang824/skm/commit/47d2f50b0deaac7fb974f4169a01a27b3b10e141))
* feat: Initialize Wails application with backend integration and frontend assets ([66be129](https://github.com/sichang824/skm/commit/66be12983ddaaf07181d704bc50748d0bb1b8e69))
* feat: enhance skill deletion process with force option and update API and UI accordingly ([3af5e00](https://github.com/sichang824/skm/commit/3af5e006d1842913105a19928c963b8bf2a0f104))
* feat: enhance skill relation model with include/exclude rules and update API and UI accordingly ([97e0ed7](https://github.com/sichang824/skm/commit/97e0ed7e96f9272b5aa032a2b59dee04a08b79cd))
* feat: add grouping functionality for skills in list view with related skills display ([eb46870](https://github.com/sichang824/skm/commit/eb46870a496d863b447a1ae57b9c002c07a41154))
* feat: add sync functionality for skills with new endpoint and UI integration ([c8013af](https://github.com/sichang824/skm/commit/c8013af8cb7df7383e25132dcd978c6edd3bc8bf))
* feat: implement skill attachment functionality with new 'attach' mode and update UI for skill relations ([eeb8770](https://github.com/sichang824/skm/commit/eeb8770121ef5a7dc101be07dc5d3c5c186aef65))
* feat: add delete skill functionality with confirmation dialog ([050c726](https://github.com/sichang824/skm/commit/050c72603bcec35fd1276ebaa0a4af46c699d33d))
* feat: implement skill attachment functionality with move and link options ([bcf9aee](https://github.com/sichang824/skm/commit/bcf9aee7fa9d6fd68adbc21341fa785993fb871c))
* feat: add seeding functionality for default providers ([477405a](https://github.com/sichang824/skm/commit/477405a717629e209d7436461002d169d1589114))
* feat: Enhance Issues, Providers, and Skills pages with improved UI and functionality ([b80186c](https://github.com/sichang824/skm/commit/b80186c99e0047e8f77694edd163a3e993d22fd7))
* feat: add Issues, NotFound, Providers, SkillDetail, and Skills pages ([70f8d5b](https://github.com/sichang824/skm/commit/70f8d5beb3c2c55beab9ae7656f602a467c8bfd9))

