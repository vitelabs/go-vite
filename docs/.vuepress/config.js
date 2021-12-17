const path = require('path');

const basePath = process.env.VUEPRESS_BASE || '/';

module.exports = {
    theme: "cosmos",
    title: "Vite: Bridging Every Blockchain in a Multi-Chain Future",
    base: basePath,

    locales: {
      "/": {
        lang: "en-US"
      },
    },

    head: [
      ['link', { rel: 'icon', href: 'https://vite.org/icon.png' }]
    ],

    plugins: [
      ["@vite/vuepress-plugin-mathjax"],
      [
        "sitemap",
        {
          hostname: "https://docs.vite.org"
        }
      ],
      [require('./plugins/code-group.js')]
    ],

    themeConfig: {
      repo: "vitelabs/go-vite",
      docsRepo: "vitelabs/go-vite",
      docsDir: 'docs',
      editLinks: true,
      custom: true,
      logo: {
        src: path.join(basePath, '/logo.svg'),
      },
      algolia: {
        id: "BH4D9OD16A",
        key: "d3bce2a3d5eff7e76467823aae0c647e",
        index: "vitelabs"
      },
      topbar: {
        banner: false
      },
      sidebar: {
        auto: true,
        nav: [
          {
            title: "Resources",
            children: [
              {
                title: "Tech Articles",
                path: "/articles"
              }
            ]
          },
          {
            title: "SDK",
            children: [
              {
                title: "Vite.js",
                path: "https://docs.vite.org/vite.js/"
              },
              {
                title: "Vite JAVA SDK",
                path: "https://docs.vite.org/vitej/"
              }
            ]
          }
        ]
      },
      footer: {
        logo: path.join(basePath, "/logo.svg"),
        textLink: {
          text: "vite.org",
          url: "https://vite.org"
        },
        services: [
          {
            service: "twitter",
            url: "https://twitter.com/vitelabs"
          },
          {
            service: "medium",
            url: "https://medium.com/vitelabs"
          },
          {
            service: "telegram",
            url: "https://t.me/vite_ann"
          },
          {
            service: "discord",
            url: "https://discord.com/invite/CsVY76q"
          },
          {
            service: "github",
            url: "https://github.com/vitelabs"
          }
        ],
        smallprint: `© ${new Date().getFullYear()} Vite Labs.`,
        links: [
          {
            title: "Community",
            children: [
              {
                title: "Blog",
                url: "https://medium.com/vitelabs"
              },
            ]
          },
          {
            title: "Community",
            children: [
              {
                title: "Contributing to the docs",
                url: "https://github.com/vitelabs/go-vite/blob/master/docs/DOCS_README.md"
              },
            ]
          }
        ]
      }
    },
};
