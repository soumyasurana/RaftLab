FROM node:22-alpine AS deps

WORKDIR /app

COPY package*.json ./
COPY dashboard/package*.json ./dashboard/

RUN npm ci

FROM node:22-alpine AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

RUN npm run build --workspace dashboard

FROM node:22-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=builder /app/dashboard/.next/standalone ./
COPY --from=builder /app/dashboard/.next/static ./dashboard/.next/static

EXPOSE 3000

CMD ["node", "server.js"]