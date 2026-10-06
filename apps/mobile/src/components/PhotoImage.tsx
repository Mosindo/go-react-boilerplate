import React, { useEffect, useRef, useState } from "react";
import { ActivityIndicator, Image, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { getFreshAccessToken } from "../api/client";
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
  const retriedFor = useRef<string | null>(null);
  const authorization = source?.headers.Authorization;

  // A new token (after a refresh) or a new photo gets a fresh attempt.
  useEffect(() => {
    setState("loading");
  }, [authorization, path]);

  const onError = () => {
    setState("error");
    if (path && retriedFor.current !== path) {
      // Most likely an expired access token: refresh once; the new token re-renders this image.
      retriedFor.current = path;
      void getFreshAccessToken().catch(() => undefined);
    }
  };

  return (
    <View style={[styles.root, { backgroundColor: colors.surfaceSubtle }, style]}>
      {source && state !== "error" ? (
        <Image
          accessibilityLabel={accessibilityLabel}
          accessible={Boolean(accessibilityLabel)}
          onError={onError}
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
