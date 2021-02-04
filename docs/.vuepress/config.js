const path = require('path');

const basePath = process.env.VUEPRESS_BASE || '/';

module.exports = {
    theme: "cosmos",
    title: "Vite Core",
    base: basePath,

    locales: {
      "/": {
        lang: "en-US"
      },
    },

    themeConfig: {
      repo: "vitelabs/go-vite",
      docsRepo: "vitelabs/go-vite",
      docsDir: 'docs',
      editLinks: true,
      custom: true,
      logo: {
        src: path.join(basePath, '/logo.svg'),
      },
      // algolia: {
      //   key: "fe006d1336f2a85d144fdfaf4a089378",
      //   index: "vite_labs"
      // },
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
                title: "Simple Introduction to Vite",
                path: "/start"
              },
              {
                title: "Tech Articles",
                path: "/articles"
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
          }
        ]
      }
    },
};