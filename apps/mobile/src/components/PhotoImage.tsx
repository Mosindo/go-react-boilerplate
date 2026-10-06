import React, { useState } from "react";
import { ActivityIndicator, Image, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useAuthedImage } from "../shared/useAuthedImage";
import { useTheme } from "../shared/ui";

type PhotoImageProps = {
  path: string | null | undefined;
  style?: StyleProp<ViewStyle>;
  resizeMode?: "cover" | "contain";
  accessibilityLabel?: string;
};

/** Private photo with a calm placeholder while loading or when it fails. */
export function PhotoImage({ path, style, resizeMode = "cover", accessibilityLabel }: PhotoImageProps) {
  const { colors } = useTheme();
  const source = useAuthedImage(path);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");

  return (
    <View style={[styles.root, { backgroundColor: colors.surfaceSubtle }, style]}>
      {source && state !== "error" ? (
        <Image
          accessibilityLabel={accessibilityLabel}
          accessible={Boolean(accessibilityLabel)}
          onError={() => setState("error")}
          onLoad={() => setState("ready")}
          resizeMode={resizeMode}
          source={source}
          style={StyleSheet.absoluteFill}
        />
      ) : null}
      {state === "loading" && source ? <ActivityIndicator color={colors.primary} /> : null}
      {!source || state === "error" ? <Ionicons color={colors.textSubtle} name="person" size={48} /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { alignItems: "center", justifyContent: "center", overflow: "hidden" }
});
