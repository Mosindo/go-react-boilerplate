import React, { useState } from "react";
import { Image, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { resolveMediaUrl } from "../api/platform";
import { Text, colors } from "../shared/ui";

type PhotoImageProps = {
  path: string | null | undefined;
  style?: StyleProp<ViewStyle>;
  /** Shown when there is no photo or it fails to load. */
  fallbackLabel?: string;
  resizeMode?: "cover" | "contain";
  testID?: string;
};

export function PhotoImage({ path, style, fallbackLabel, resizeMode = "cover", testID }: PhotoImageProps) {
  const [failed, setFailed] = useState(false);
  const uri = resolveMediaUrl(path);
  return (
    <View style={[styles.frame, style]} testID={testID}>
      {uri && !failed ? (
        <Image accessibilityIgnoresInvertColors onError={() => setFailed(true)} resizeMode={resizeMode} source={{ uri }} style={styles.image} />
      ) : (
        <View style={styles.fallback}>
          <Text style={styles.initial} tone="muted" weight="bold">
            {fallbackLabel ? fallbackLabel.slice(0, 1).toUpperCase() : "?"}
          </Text>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  frame: { backgroundColor: colors.surfaceSubtle, overflow: "hidden" },
  image: { width: "100%", height: "100%" },
  fallback: { flex: 1, alignItems: "center", justifyContent: "center", backgroundColor: colors.primarySoft },
  initial: { fontSize: 42, lineHeight: 48 }
});
