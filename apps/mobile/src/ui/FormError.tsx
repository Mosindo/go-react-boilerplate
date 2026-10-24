import React from "react";
import { useTheme } from "../theme/theme";
import { Text } from "./Text";

export function FormError({
  message,
  testID,
}: {
  message: string | null | undefined;
  testID?: string;
}) {
  const { colors } = useTheme();
  if (!message) return null;
  return (
    <Text testID={testID} color={colors.danger} accessibilityRole="alert">
      {message}
    </Text>
  );
}
