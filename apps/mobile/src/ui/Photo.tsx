import React from "react";
import { View, type ImageStyle, type StyleProp, type ViewStyle } from "react-native";
import { Image } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import { mediaUrl } from "../api/config";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  url?: string | null;
  style?: StyleProp<ImageStyle>;
  rounded?: number;
  accessibilityLabel?: string;
  priority?: "low" | "normal" | "high";
};

/** Cached remote photo (memory + disk) with a neutral placeholder. */
export function Photo({ url, style, rounded = 0, accessibilityLabel, priority = "normal" }: Props) {
  const { colors } = useTheme();
  if (!url) {
    return (
      <View
        style={[
          {
            backgroundColor: colors.surfaceSunken,
            alignItems: "center",
            justifyContent: "center",
            borderRadius: rounded
          },
          style as StyleProp<ViewStyle>
        ]}
      >
        <Ionicons name="person-outline" size={28} color={colors.textSubtle} />
      </View>
    );
  }
  return (
    <Image
      source={{ uri: mediaUrl(url) }}
      style={[{ borderRadius: rounded, backgroundColor: colors.surfaceSunken }, style]}
      contentFit="cover"
      cachePolicy="memory-disk"
      transition={180}
      priority={priority}
      accessibilityLabel={accessibilityLabel}
    />
  );
}

export function Avatar({ url, size = 52, label }: { url?: string | null; size?: number; label?: string }) {
  return <Photo url={url} rounded={size / 2} style={{ width: size, height: size }} accessibilityLabel={label} />;
}
