"use client";

import { FormEvent, useState } from "react";

type ShortenResponse = {
  shortCode: string;
  shortUrl: string;
  expiresAt: string;
};

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

export default function Home() {
  const [url, setUrl] = useState("");
  const [result, setResult] = useState<ShortenResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setResult(null);
    setLoading(true);

    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/shorten`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ url }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || "Cannot shorten URL");
      }

      setResult(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Something went wrong";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <main>
      <section className="card">
        <h1>Go URL Shortener</h1>
        <p>
          Portfolio project with Golang, Hexagonal Clean Architecture,
          PostgreSQL, Redis and Next.js
        </p>

        <form className="form" onSubmit={handleSubmit}>
          <input
            type="url"
            placeholder="https://example.com"
            value={url}
            onChange={(event) => setUrl(event.target.value)}
            required
          />
          <button type="submit" disabled={loading}>
            {loading ? "Creating..." : "Shorten"}
          </button>
        </form>

        {error && <div className="error">{error}</div>}

        {result && (
          <div className="result">
            <strong>Short URL</strong>
            <p>
              <a href={result.shortUrl} target="_blank" rel="noreferrer">
                {result.shortUrl}
              </a>
            </p>
            <p className="meta">Expires at: {result.expiresAt}</p>
          </div>
        )}
      </section>
    </main>
  );
}
