import React, { useState } from "react";
import { Image, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";

import { absoluteUrl } from "../api/client";
import type { Photo } from "../api/types";
import { useTheme } from "../theme";
import { Text } from "./Text";

export interface PhotoImageProps {
  photo?: Photo;
  /** Used for the fallback initial when there is no photo or it fails to load. */
  name?: string;
  style?: StyleProp<ViewStyle>;
  round?: boolean;
  size?: number;
}

export function PhotoImage({ photo, name, style, round, size }: PhotoImageProps) {
  const { colors } = useTheme();
  const [failed, setFailed] = useState(false);
  const dimension = size ? { width: size, height: size } : null;

  return (
    <View
      style={[
        styles.box,
        { backgroundColor: colors.primarySoft },
        dimension,
        round && { borderRadius: (size ?? 48) / 2 },
        style,
      ]}
    >
      {photo && !failed ? (
        <Image
          accessibilityLabel={name ? `Photo de ${name}` : "Photo"}
          source={{ uri: absoluteUrl(photo.url) }}
          style={StyleSheet.absoluteFill}
          resizeMode="cover"
          onError={() => setFailed(true)}
        />
      ) : (
        <Text variant="title" tone="primary">
          {(name ?? "?").slice(0, 1).toUpperCase()}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  box: { overflow: "hidden", alignItems: "center", justifyContent: "center" },
});
