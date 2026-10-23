import React, { useEffect, useRef } from "react";
import { Animated, View, type DimensionValue, type ViewStyle } from "react-native";
import { t } from "../i18n/fr";
import { useTheme } from "../theme/ThemeProvider";

type Props = { width?: DimensionValue; height?: number; radius?: number; style?: ViewStyle };

export function Skeleton({ width = "100%", height = 16, radius = 8, style }: Props) {
  const { colors } = useTheme();
  const pulse = useRef(new Animated.Value(0.5)).current;
  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(pulse, { toValue: 1, duration: 800, useNativeDriver: true }),
        Animated.timing(pulse, { toValue: 0.5, duration: 800, useNativeDriver: true })
      ])
    );
    loop.start();
    return () => loop.stop();
  }, [pulse]);
  return (
    <Animated.View
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
      style={[
        { width, height, borderRadius: radius, backgroundColor: colors.skeleton, opacity: pulse },
        style
      ]}
    />
  );
}

/** Placeholder rows for list screens. */
export function ListSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <View style={{ gap: 16, paddingVertical: 8 }} accessibilityLabel={t.common.loading}>
      {Array.from({ length: rows }, (_, index) => (
        <View key={index} style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <Skeleton width={56} height={56} radius={28} />
          <View style={{ flex: 1, gap: 8 }}>
            <Skeleton width="45%" height={16} />
            <Skeleton width="80%" height={13} />
          </View>
        </View>
      ))}
    </View>
  );
}
