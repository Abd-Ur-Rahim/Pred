"use client";

import { AssetDetail } from "@/components/ui/asset-detail";

interface MotorsProps {
  onBack?: () => void;
}

export function MotorsView({ onBack }: MotorsProps) {
  return <AssetDetail onBack={onBack} />;
}
