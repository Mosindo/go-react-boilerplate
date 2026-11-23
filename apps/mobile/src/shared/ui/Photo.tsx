import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image, type ImageStyle } from "expo-image";
import { mediaUrl } from "../../api/client";
import { Text } from "./Text";
import { colors } from "./tokens";

type PhotoProps = {
  url?: string | null;
  style?: StyleProp<ImageStyle>;
  placeholderLabel?: string;
  accessibilityLabel?: string;
  priority?: "low" | "normal" | "high";
};

/** Protected profile photo rendered from a signed API URL with disk caching. */
export function Photo({ accessibilityLabel, placeholderLabel, priority = "normal", style, url }: PhotoProps) {
  if (!url) {
    return (
      <View style={[styles.placeholder, style as StyleProp<ViewStyle>]}>
        {placeholderLabel ? (
          <Text tone="subtle" variant="heading">
            {placeholderLabel}
          </Text>
        ) : null}
      </View>
    );
  }
  return (
    <Image
      accessibilityLabel={accessibilityLabel}
      cachePolicy="memory-disk"
      contentFit="cover"
      priority={priority}
      recyclingKey={url.split("?")[0]}
      source={{ uri: mediaUrl(url) }}
      style={[styles.image, style]}
      transition={180}
    />
  );
}

type AvatarProps = {
  url?: string | null;
  name: string;
  size?: number;
};

export function Avatar({ name, size = 52, url }: AvatarProps) {
  return (
    <Photo
      accessibilityLabel={`Photo de ${name}`}
      placeholderLabel={name.slice(0, 1).toUpperCase()}
      style={{ width: size, height: size, borderRadius: size / 2 }}
      url={url}
    />
  );
}

const styles = StyleSheet.create({
  image: {
    backgroundColor: colors.surfaceRaised
  },
  placeholder: {
    backgroundColor: colors.surfaceRaised,
    alignItems: "center",
    justifyContent: "center"
  }
});
