"use client";

import { useSyncExternalStore } from "react";

const format = new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" });
const subscribe = () => () => {};

// the server does not know the user's timezone, so the text is only filled in the browser
export function LocalTime({ iso }: { iso: string }) {
  const text = useSyncExternalStore(
    subscribe,
    () => format.format(new Date(iso)),
    () => "",
  );
  return <time dateTime={iso}>{text}</time>;
}
