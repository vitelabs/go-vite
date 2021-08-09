## Docs Build Workflow

The documentation for the go-vite is hosted at https://docs.vite.org/go-vite

Built from the files in this (`/docs`) directory for
[master](https://github.com/vitelabs/go-vite/tree/master/docs).

### How It Works

We use github action for docs deploy. Every time you push code to github, if some changes is happen on `docs/**`. It will auto deploy to https://docs.vite.org/go-vite/

## README

The [README.md](./README.md) is also the landing page for the documentation
on the website. 

## Config.js

The [config.js](./.vuepress/config.js) generates the sidebar and Table of Contents
on the website docs. 

We have enable auto generate sidebar. So you don't need to setup sidebar manual. 

## Markdown configuration

Markdown files can contain YAML frontmatter. Several properties (all of which are optional) are used:

```
---
# title is displayed in the sidebar
title: Title of the file
# order specifies file's priority in the sidebar
order: 2
# parent is readme.md or index.md parent directory
parent:
  title: Directory title
  order: 1
---
```

Setting `order: false` removes the item (file or directory) from the sidebar. It is, however, remains accessible by means other than the sidebar. It is valid use a `readme.md` to set an order of a parent-directory and hide the file with `order: false`.

## Links

We prefer `relative path` when use link in file.

Such as:

```
[test](./README.md)
```
It works as follow:

[test](./README.md)

## Directory Structure with Images

If you have images or other assets to link, please put the assets nearly by the file. And link it with `relative path`.

Such as:

```
- test.md
- test.png
```

If one file have links many assets, Please put assets into a `dir` named with the origin file name.

For example:

```
- test.md
- test
   - test-01.png
   - test-02.png
   - test-03.png
```

If one `dir` have many files and assets. Please put all assets into a `dir` named `assets`.

For example:

```
- test1.md
- test2.md
- test3.md
- test4.md
- test5.md
- assets
   - test1
      - test1-01.png
      - test1-02.png
      - test1-03.png
   - test2
      - test2-01.png
      - test2-02.png
      - test2-03.png
   - test3
      - test3-01.png
      - test3-02.png
      - test3-03.png
```

Here is the real example: https://github.com/vitelabs/go-vite/tree/master/docs/articles

All images in `articles` directory are joined in `articles/assets` directory.

## Running Locally

Make sure you are in the `docs` directory and run the following commands:

```sh
rm -rf node_modules
```

This command will remove old version of the visual theme and required packages. This step is optional.

```sh
yarn install
```

Install the theme and all dependencies.

```sh
yarn run dev
```

To build documentation as a static website run `yarn run build`. You will find the website in `.vuepress/dist` directory.
