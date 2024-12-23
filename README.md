# Nuxt Launcher

A web-based service launcher and log viewer built with Nuxt.js.

## Features

- Launch and manage multiple services from a single interface
- Real-time service status monitoring
- Live log viewing with error tracking
- WebSocket-based communication for instant updates
- Track read/unread error logs
- Persistent log read status

## Technical Stack

- [Nuxt.js](https://nuxt.com/)
- [Pinia](https://pinia.vuejs.org/) for state management
- [TailwindCSS](https://tailwindcss.com/) for styling
- WebSocket for real-time updates

## Development

1. Install dependencies:

```bash
# npm
npm install

# pnpm
pnpm install

# yarn
yarn install

# bun
bun install
```

2. Start the development server on `http://localhost:3000`:

```bash
# npm
npm run dev

# pnpm
pnpm dev

# yarn
yarn dev

# bun
bun dev
```

## Production

Build the application for production:

```bash
# npm
npm run build

# pnpm
pnpm build

# yarn
yarn build

# bun
bun run build
```

Locally preview production build:

```bash
# npm
npm run preview

# pnpm
pnpm preview

# yarn
yarn preview

# bun
bun run preview
```

Check out the [deployment documentation](https://nuxt.com/docs/getting-started/deployment) for more information.
