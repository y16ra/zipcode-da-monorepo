"use client";

import { useCallback, useEffect, useRef, useState } from "react";

const apiBase =
  process.env.NEXT_PUBLIC_API_BASE_URL?.replace(/\/$/, "") ||
  "http://localhost:8080";

type SearchBody = {
  zipcode: string;
  page?: number;
  limit?: number;
  choikitype?: number;
  searchtype?: number;
  ec_uid?: string;
};

const DEBOUNCE_MS = 600;

function formatZipcode(value: string): string {
  const digits = value.replace(/\D/g, "").slice(0, 7);
  if (digits.length > 3) {
    return `${digits.slice(0, 3)}-${digits.slice(3)}`;
  }
  return digits;
}

function validateZipcode(value: string): string | null {
  const digits = value.replace(/\D/g, "");
  if (digits.length === 0) return "郵便番号を入力してください";
  if (digits.length < 3) return "3桁以上入力してください";
  if (digits.length > 7) return "7桁以内で入力してください";
  return null;
}

export default function HomePage() {
  const [zipcode, setZipcode] = useState("");
  const [zipcodeError, setZipcodeError] = useState<string | null>(null);
  const [page, setPage] = useState("1");
  const [limit, setLimit] = useState("100");
  const [choikitype, setChoikitype] = useState("1");
  const [searchtype, setSearchtype] = useState("1");
  const [ecUid, setEcUid] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [resultJson, setResultJson] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const search = useCallback(
    async (code: string) => {
      const validationError = validateZipcode(code);
      if (validationError) return;

      setError(null);
      setResultJson(null);
      setLoading(true);
      try {
        const normalizedCode = code.replace(/\D/g, "");
        const body: SearchBody = { zipcode: normalizedCode };
        const p = parseInt(page, 10);
        const l = parseInt(limit, 10);
        const c = parseInt(choikitype, 10);
        const s = parseInt(searchtype, 10);
        if (!Number.isNaN(p)) body.page = p;
        if (!Number.isNaN(l)) body.limit = l;
        if (!Number.isNaN(c)) body.choikitype = c;
        if (!Number.isNaN(s)) body.searchtype = s;
        if (ecUid.trim()) body.ec_uid = ecUid.trim();

        const res = await fetch(`${apiBase}/api/v1/search/zipcode`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
        const text = await res.text();
        if (!res.ok) {
          setError(`${res.status} ${res.statusText}\n${text}`);
          return;
        }
        try {
          setResultJson(JSON.stringify(JSON.parse(text), null, 2));
        } catch {
          setResultJson(text);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setLoading(false);
      }
    },
    [page, limit, choikitype, searchtype, ecUid],
  );

  const onZipcodeChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const formatted = formatZipcode(e.target.value);
      setZipcode(formatted);

      const err = validateZipcode(formatted);
      setZipcodeError(err);

      if (debounceRef.current) clearTimeout(debounceRef.current);

      const digits = formatted.replace(/\D/g, "");
      if (!err && digits.length === 7) {
        debounceRef.current = setTimeout(() => {
          search(formatted);
        }, DEBOUNCE_MS);
      }
    },
    [search],
  );

  const onSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const err = validateZipcode(zipcode);
      setZipcodeError(err);
      if (err) return;
      if (debounceRef.current) clearTimeout(debounceRef.current);
      await search(zipcode);
    },
    [zipcode, search],
  );

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  const digits = zipcode.replace(/\D/g, "");
  const isComplete = digits.length === 7;

  return (
    <main>
      <h1>郵便番号検索</h1>
      <p className="lead">
        バックエンドの <code>POST /api/v1/search/zipcode</code> を呼び出し、日本郵便
        searchcode API の結果を表示します。
      </p>

      <form onSubmit={onSubmit}>
        <label>
          郵便番号（3〜7桁、ハイフン可）
          <input
            value={zipcode}
            onChange={onZipcodeChange}
            inputMode="numeric"
            autoComplete="postal-code"
            placeholder="例: 100-0001"
            aria-invalid={zipcodeError !== null}
            required
          />
          {zipcodeError && (
            <span className="field-error">{zipcodeError}</span>
          )}
          {!zipcodeError && isComplete && (
            <span className="field-hint">7桁入力完了 — 自動検索します</span>
          )}
        </label>
        <div className="grid2">
          <label>
            page（省略時 1）
            <input value={page} onChange={(e) => setPage(e.target.value)} />
          </label>
          <label>
            limit（1〜1000、省略時 1000）
            <input value={limit} onChange={(e) => setLimit(e.target.value)} />
          </label>
          <label>
            choikitype（1: 括弧なし / 2: 括弧あり）
            <select
              value={choikitype}
              onChange={(e) => setChoikitype(e.target.value)}
            >
              <option value="1">1</option>
              <option value="2">2</option>
            </select>
          </label>
          <label>
            searchtype（1: 全対象 / 2: 事業所個別郵便番号除外）
            <select
              value={searchtype}
              onChange={(e) => setSearchtype(e.target.value)}
            >
              <option value="1">1</option>
              <option value="2">2</option>
            </select>
          </label>
        </div>
        <label>
          ec_uid（任意）
          <input
            value={ecUid}
            onChange={(e) => setEcUid(e.target.value)}
            placeholder="未指定ならサーバ環境変数 JAPANPOST_EC_UID を使用"
          />
        </label>
        <button type="submit" disabled={loading || !!zipcodeError}>
          {loading ? "検索中…" : "検索"}
        </button>
      </form>

      {error && <div className="error">{error}</div>}

      {resultJson && (
        <section className="results">
          <h2>レスポンス</h2>
          <pre>{resultJson}</pre>
        </section>
      )}
    </main>
  );
}
