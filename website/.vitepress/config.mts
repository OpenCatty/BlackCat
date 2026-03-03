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
      { text: 'CLI', link: '/cli/onboard' },
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
          { text: 'Installation', link: '/installation' },
        ]
      },
      {
        text: 'CLI Reference',
        items: [
          { text: 'onboard', link: '/cli/onboard' },
          { text: 'configure', link: '/cli/configure' },
          { text: 'start / stop / restart', link: '/cli/start' },
          { text: 'status', link: '/cli/status' },
          { text: 'channels', link: '/cli/channels' },
          { text: 'doctor', link: '/cli/doctor' },
          { text: 'vault', link: '/cli/vault' },
          { text: 'health', link: '/cli/health' },
          { text: 'uninstall', link: '/cli/uninstall' },
        ]
      },
      {
        text: 'Channels',
        items: [
          { text: 'WhatsApp', link: '/channels/whatsapp' },
          { text: 'Telegram', link: '/channels/telegram' },
          { text: 'Discord', link: '/channels/discord' },
        ]
      },
      {
        text: 'Providers',
        items: [
          { text: 'Overview', link: '/providers' },
          { text: 'OpenAI', link: '/providers/openai' },
          { text: 'Anthropic', link: '/providers/anthropic' },
          { text: 'GitHub Copilot', link: '/providers/copilot' },
          { text: 'Antigravity', link: '/providers/antigravity' },
          { text: 'Google Gemini', link: '/providers/gemini' },
          { text: 'OpenRouter', link: '/providers/openrouter' },
          { text: 'Ollama', link: '/providers/ollama' },
          { text: 'Zen Coding Plan', link: '/providers/zen' },
        ]
      },
      {
        text: 'Concepts',
        items: [
          { text: 'Architecture', link: '/concepts/architecture' },
          { text: 'Dashboard', link: '/concepts/dashboard' },
          { text: 'Hooks & Rules', link: '/concepts/hooks' },
          { text: 'OAuth', link: '/concepts/oauth' },
        ]
      },
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
