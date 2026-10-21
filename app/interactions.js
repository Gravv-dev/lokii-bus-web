"use client";

import { useEffect } from "react";

export default function Interactions() {
  useEffect(() => {
    import("../js/main.js").then(({ initPage }) => initPage());
  }, []);

  return null;
}
