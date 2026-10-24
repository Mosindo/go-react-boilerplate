import React from "react";
import { View } from "react-native";
import { useTheme } from "../theme/theme";
import { Photo } from "./Photo";
import { Text } from "./Text";

type Props = { photoId?: string | null; name: string; size?: number };

export function Avatar({ photoId, name, size = 48 }: Props) {
  const { colors } = useTheme();
  const base = { width: size, height: size, borderRadius: size / 2, overflow: "hidden" as const };
  if (photoId) {
    return (
      <View style={base} accessible accessibilityLabel={`Photo de ${name}`}>
        <Photo photoId={photoId} size="thumb" style={{ width: size, height: size }} />
      </View>
    );
  }
  return (
    <View
      style={[
        base,
        { backgroundColor: colors.surfaceAlt, alignItems: "center", justifyContent: "center" },
      ]}
    >
      <Text variant="heading" color={colors.primary}>
        {name.slice(0, 1).toUpperCase()}
      </Text>
    </View>
  );
}
