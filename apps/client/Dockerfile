FROM oven/bun:1 AS build
WORKDIR /app
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile
COPY . .
RUN bun run build

FROM oven/bun:1-slim
RUN groupadd -r journal && useradd -r -g journal journal
WORKDIR /app
COPY --from=build /app/build ./build
COPY --from=build /app/package.json ./
RUN chown -R journal:journal /app
USER journal
ENV PORT=3000
ENV ORIGIN=http://localhost:3000
EXPOSE 3000
CMD ["bun", "./build/index.js"]
