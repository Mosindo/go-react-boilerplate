import React from "react";
import { Pressable, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Text } from "../ui/Text";
import { controls } from "../ui/tokens";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";
import { formatCount } from "../../lib/format";

export type TabItem<K extends string> = {
  key: K;
  label: string;
  glyph: string;
  badge?: number;
  testID?: string;
};

type TabBarProps<K extends string> = {
  items: readonly TabItem<K>[];
  active: K;
  onChange: (key: K) => void;
};

const makeStyles = (t: Theme) => ({
  bar: {
    flexDirection: "row" as const,
    backgroundColor: t.colors.backgroundElevated,
    borderTopWidth: 1,
    borderTopColor: t.colors.border,
    paddingTop: t.spacing.xxs
  },
  tab: {
    flex: 1,
    minHeight: controls.minTarget + 12,
    alignItems: "center" as const,
    justifyContent: "center" as const,
    gap: 2
  },
  glyphWrap: { minWidth: 28, alignItems: "center" as const },
  badge: {
    position: "absolute" as const,
    top: -4,
    right: -10,
    minWidth: 18,
    height: 18,
    borderRadius: 9,
    paddingHorizontal: 4,
    backgroundColor: t.colors.primary,
    alignItems: "center" as const,
    justifyContent: "center" as const
  }
});

export function TabBar<K extends string>({ active, items, onChange }: TabBarProps<K>) {
  const styles = useThemedStyles(makeStyles);
  const insets = useSafeAreaInsets();
  return (
    <View
      accessibilityRole="tablist"
      style={[styles.bar, { paddingBottom: Math.max(insets.bottom, 6) }]}
    >
      {items.map((item) => {
        const selected = item.key === active;
        const badge = formatCount(item.badge ?? 0);
        return (
          <Pressable
            accessibilityLabel={badge ? `${item.label}, ${item.badge} unread` : item.label}
            accessibilityRole="tab"
            accessibilityState={{ selected }}
            key={item.key}
            onPress={() => onChange(item.key)}
            style={styles.tab}
            testID={item.testID ?? `tab-${item.key}`}
          >
            <View style={styles.glyphWrap}>
              <Text
                importantForAccessibility="no"
                tone={selected ? "primary" : "muted"}
                variant="heading"
                weight="bold"
              >
                {item.glyph}
              </Text>
              {badge ? (
                <View style={styles.badge}>
                  <Text
                    importantForAccessibility="no"
                    tone="inverse"
                    variant="caption"
                    weight="bold"
                  >
                    {badge}
                  </Text>
                </View>
              ) : null}
            </View>
            <Text
              tone={selected ? "primary" : "muted"}
              variant="caption"
              weight={selected ? "bold" : "medium"}
            >
              {item.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}
