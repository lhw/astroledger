# Stage 1: Build the SvelteKit app
FROM node:22-alpine AS builder

WORKDIR /app

# Cache npm install separately from source
COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# adapter-node produces a standalone Node.js server in /build
RUN npm run build

# Stage 2: Minimal runtime image
FROM node:22-alpine

WORKDIR /app

# Copy the built app and only the runtime deps
COPY --from=builder /app/build build/
COPY --from=builder /app/package.json .

# adapter-node bundles everything needed into build/
# No additional npm install needed

EXPOSE 3000

# ORIGIN must be set at runtime: the external HTTPS URL of the site
# e.g. ORIGIN=https://astroledger.example.com
ENV NODE_ENV=production

CMD ["node", "build/index.js"]
