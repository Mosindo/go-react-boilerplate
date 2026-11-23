import React from "react";
import { ActivityIndicator } from "react-native";
import { colors } from "./tokens";

export function Loader({ size = "small" }: { size?: "small" | "large" }) {
  return <ActivityIndicator accessibilityLabel="Chargement" color={colors.primary} size={size} />;
}
