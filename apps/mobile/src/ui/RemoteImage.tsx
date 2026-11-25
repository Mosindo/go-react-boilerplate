import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image, type ImageStyle } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import { absoluteUrl } from "../api/config";
import { colors } from "../theme";

type RemoteImageProps = {
  path: string | null | undefined;
  style?: StyleProp<ImageStyle>;
  placeholderStyle?: StyleProp<ViewStyle>;
  accessibilityLabel?: string;
  priority?: "low" | "normal" | "high";
};

/** Displays an API photo with memory/disk caching and a quiet placeholder. */
export function RemoteImage({ path, style, placeholderStyle, accessibilityLabel, priority = "normal" }: RemoteImageProps) {
  if (!path) {
    return (
      <View style={[styles.placeholder, placeholderStyle, style as StyleProp<ViewStyle>]} accessibilityLabel={accessibilityLabel}>
        <Ionicons name="person" size={32} color={colors.textSubtle} />
      </View>
    );
  }
  return (
    <Image
      source={{ uri: absoluteUrl(path) }}
      style={[styles.image, style]}
      contentFit="cover"
      transition={180}
      cachePolicy="memory-disk"
      priority={priority}
      accessibilityLabel={accessibilityLabel}
      recyclingKey={path}
    />
  );
}

const styles = StyleSheet.create({
  image: {
    backgroundColor: colors.surfaceHigh
  },
  placeholder: {
    backgroundColor: colors.surfaceHigh,
    alignItems: "center",
    justifyContent: "center"
  }
});
