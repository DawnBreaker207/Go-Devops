# syntax=docker/dockerfile:1
# ---------- deps stage ----------
FROM node:22-alpine AS deps

WORKDIR /app

COPY package.json package-lock.json ./
# Plain RUN (no cache mount): node_modules must live IN the image because
# the test stage below executes at container runtime, not at build time.
RUN npm ci

# ---------- test stage (CI) ----------
FROM deps AS test

COPY . .

CMD ["sh", "-c", "npx tsc --noEmit -p tsconfig.app.json && npm run lint && npm test"]

# ---------- build stage ----------
FROM deps AS builder

COPY . .

ARG VITE_API_BASE_URL
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

RUN npm run build

# ---------- runtime stage ----------
FROM nginx:1.27-alpine AS runner

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget -qO- http://localhost/ >/dev/null || exit 1

CMD ["nginx", "-g", "daemon off;"]
