import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../shared/ui/theme";

const BANDS = 8;

/**
 * Bottom-anchored dark gradient built from stacked translucent bands (no gradient dependency).
 * `height` is the total height of the scrim; opacity grows towards the bottom.
 */
export function Scrim({ height, style }: { height: number; style?: StyleProp<ViewStyle> }) {
  const { colors } = useTheme();
  const bandHeight = height / BANDS;
  return (
    <View pointerEvents="none" style={[styles.wrap, { height }, style]}>
      {Array.from({ length: BANDS }, (_, i) => (
        <View
          key={i}
          style={{
            height: bandHeight,
            backgroundColor: colors.photoScrim,
            opacity: (i + 1) / BANDS
          }}
        />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: 0, right: 0, bottom: 0 }
});
