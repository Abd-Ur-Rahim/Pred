from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import joblib
import pandas as pd


@dataclass
class ModelArtifacts:
    model: Any
    feature_columns: list[str]
    threshold_low: float
    threshold_high: float


class BearingAnomalyPredictor:
    def __init__(
        self,
        model_path: str | Path,
        feature_columns_path: str | Path,
        thresholds_path: str | Path,
        model_name: str = "vibration_isolation_forest",
        model_version: str = "v1",
    ) -> None:
        self.model_name = model_name
        self.model_version = model_version
        self.artifacts = self._load_artifacts(
            Path(model_path),
            Path(feature_columns_path),
            Path(thresholds_path),
        )

    @staticmethod
    def _load_artifacts(
        model_path: Path,
        feature_columns_path: Path,
        thresholds_path: Path,
    ) -> ModelArtifacts:
        model = joblib.load(model_path)

        with feature_columns_path.open("r", encoding="utf-8") as feature_file:
            feature_columns = json.load(feature_file)

        with thresholds_path.open("r", encoding="utf-8") as threshold_file:
            thresholds = json.load(threshold_file)

        threshold_low = float(thresholds["risk_score_quantile_60"])
        threshold_high = float(thresholds["risk_score_quantile_85"])

        return ModelArtifacts(
            model=model,
            feature_columns=feature_columns,
            threshold_low=threshold_low,
            threshold_high=threshold_high,
        )

    def _validate_and_prepare_input(self, feature_row: dict[str, Any]) -> pd.DataFrame:
        input_columns = set(feature_row.keys())
        expected_columns = set(self.artifacts.feature_columns)

        missing_columns = sorted(expected_columns - input_columns)
        extra_columns = sorted(input_columns - expected_columns)

        if missing_columns or extra_columns:
            details = {
                "missing_columns": missing_columns,
                "extra_columns": extra_columns,
            }
            raise ValueError(f"Input feature columns do not match training columns: {details}")

        ordered_row = {column: feature_row[column] for column in self.artifacts.feature_columns}
        return pd.DataFrame([ordered_row], columns=self.artifacts.feature_columns)

    def _score_to_status(self, anomaly_score: float) -> str:
        if anomaly_score <= self.artifacts.threshold_low:
            return "normal"
        if anomaly_score <= self.artifacts.threshold_high:
            return "warning"
        return "critical"

    def predict(
        self,
        feature_row: dict[str, Any],
        device_id: str,
        asset_id: str,
    ) -> dict[str, Any]:
        input_df = self._validate_and_prepare_input(feature_row)

        anomaly_score = float(-self.artifacts.model.decision_function(input_df)[0])
        predicted_status = self._score_to_status(anomaly_score)

        return {
            "device_id": device_id,
            "asset_id": asset_id,
            "model_name": self.model_name,
            "model_version": self.model_version,
            "anomaly_score": anomaly_score,
            "predicted_status": predicted_status,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }
