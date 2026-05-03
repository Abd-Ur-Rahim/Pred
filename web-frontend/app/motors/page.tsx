"use client";

import { MotorsView } from "@/components/views/motors";

export default function MotorsPage() {
  return <MotorsView onBack={() => window.history.back()} />;
}
