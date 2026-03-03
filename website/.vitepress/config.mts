import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'BlackCat',
  description: 'AI Agent for OpenCode via Messaging Channels',
  base: '/BlackCat/',

  head: [['link', { rel: 'icon', href: '/BlackCat/favicon.ico' }]],

  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: 'Guide', link: '/getting-started' },
      { text: 'Config', link: '/configuration' },
      { text: 'Providers', link: '/providers' },
      {
        text: 'GitHub',
        link: 'https://github.com/startower-observability/BlackCat'
      }
    ],

    sidebar: [
      {
        text: 'Getting Started',
        items: [
          { text: 'Introduction', link: '/' },
          { text: 'Quick Start', link: '/getting-started' },
          { text: 'Configuration', link: '/configuration' }
        ]
      },
      {
        text: 'Providers',
        items: [
          { text: 'LLM Providers', link: '/providers' },
          { text: 'OAuth Setup', link: '/oauth' },
          { text: 'Zen Coding Plan', link: '/zen-plan' }
        ]
      },
      {
        text: 'Features',
        items: [
          { text: 'CLI Configure', link: '/configure-cli' },
          { text: 'Dashboard', link: '/dashboard' },
          { text: 'Hooks & Rules', link: '/hooks-and-rules' }
        ]
      },
      {
        text: 'Reference',
        items: [{ text: 'Architecture', link: '/architecture' }]
      }
    ],

    socialLinks: [
      {
        icon: 'github',
        link: 'https://github.com/startower-observability/BlackCat'
      }
    ],

    search: {
      provider: 'local'
    },

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © StarTower'
    }
  }
})
