# syntax=docker/dockerfile:1.6

FROM node:24-alpine AS base

ENV PNPM_HOME=/usr/local/share/pnpm

RUN corepack enable pnpm && \
    apk add --no-cache libc6-compat netcat-openbsd

WORKDIR /app

FROM base AS deps

# Configure npm for better network reliability
ENV NPM_CONFIG_FETCH_TIMEOUT=300000
ENV NPM_CONFIG_FETCH_RETRIES=5
ENV NPM_CONFIG_FETCH_RETRY_FACTOR=10
ENV NPM_CONFIG_FETCH_RETRY_MINTIMEOUT=10000

COPY package*.json ./
RUN --mount=type=cache,id=cache-npm-cache,target=/root/.npm \
    npm ci || (sleep 5 && npm ci) || (sleep 10 && npm ci)

FROM base AS builder

# Configure npm for better network reliability
ENV NPM_CONFIG_FETCH_TIMEOUT=300000
ENV NPM_CONFIG_FETCH_RETRIES=5
ENV NPM_CONFIG_FETCH_RETRY_FACTOR=10
ENV NPM_CONFIG_FETCH_RETRY_MINTIMEOUT=10000

COPY --from=deps /app/node_modules ./node_modules
COPY . .

RUN npm run build

FROM base AS runner

ENV NODE_ENV=production

# Configure npm for better network reliability
ENV NPM_CONFIG_FETCH_TIMEOUT=300000
ENV NPM_CONFIG_FETCH_RETRIES=5
ENV NPM_CONFIG_FETCH_RETRY_FACTOR=10
ENV NPM_CONFIG_FETCH_RETRY_MINTIMEOUT=10000

WORKDIR /app

COPY package*.json ./
RUN --mount=type=cache,id=cache-npm-cache,target=/root/.npm \
    npm ci --omit=dev || (sleep 5 && npm ci --omit=dev) || (sleep 10 && npm ci --omit=dev)

COPY --from=builder /app/dist ./dist
COPY docker-entrypoint.sh ./docker-entrypoint.sh

RUN chmod +x docker-entrypoint.sh && chown -R node:node /app

USER node

EXPOSE 3000

ENTRYPOINT ["./docker-entrypoint.sh"]

