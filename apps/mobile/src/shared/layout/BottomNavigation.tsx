import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { BottomTabBarProps } from "@react-navigation/bottom-tabs";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Text, colors, radii, spacing } from "../ui";

const LABELS: Record<string, { label: string; glyph: string }> = {
  Discover: { label: "Découvrir", glyph: "♥" },
  Messages: { label: "Messages", glyph: "✉" },
  Notifications: { label: "Activité", glyph: "★" },
  Me: { label: "Profil", glyph: "☺" }
};

export type TabBadges = Partial<Record<string, number>>;

type BottomNavigationProps = BottomTabBarProps & { badges: TabBadges };

/** Custom tab bar: large touch targets, count badges, labels always visible. */
export function BottomNavigation({ state, navigation, badges }: BottomNavigationProps) {
  const insets = useSafeAreaInsets();
  return (
    <View accessibilityRole="tablist" style={[styles.bar, { paddingBottom: Math.max(insets.bottom, spacing.sm) }]}>
      {state.routes.map((route, index) => {
        const focused = state.index === index;
        const meta = LABELS[route.name] ?? { label: route.name, glyph: "•" };
        const count = badges[route.name] ?? 0;
        return (
          <Pressable
            accessibilityLabel={count > 0 ? `${meta.label}, ${count} nouveaux` : meta.label}
            accessibilityRole="tab"
            accessibilityState={{ selected: focused }}
            key={route.key}
            onPress={() => {
              const event = navigation.emit({ type: "tabPress", target: route.key, canPreventDefault: true });
              if (!focused && !event.defaultPrevented) {
                navigation.navigate(route.name);
              }
            }}
            style={styles.tab}
            testID={`tab-${route.name}`}
          >
            <View>
              <Text style={[styles.glyph, { color: focused ? colors.primary : colors.textSubtle }]}>{meta.glyph}</Text>
              {count > 0 ? (
                <View style={styles.badge}>
                  <Text style={styles.badgeText} tone="inverse" weight="bold">
                    {count > 9 ? "9+" : count}
                  </Text>
                </View>
              ) : null}
            </View>
            <Text tone={focused ? "primary" : "muted"} variant="caption" weight={focused ? "bold" : "medium"}>
              {meta.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  bar: {
    flexDirection: "row",
    backgroundColor: colors.backgroundElevated,
    borderTopWidth: 1,
    borderTopColor: colors.border,
    paddingTop: spacing.sm
  },
  tab: { flex: 1, alignItems: "center", gap: 2, minHeight: 48, justifyContent: "center" },
  glyph: { fontSize: 22, lineHeight: 26 },
  badge: {
    position: "absolute",
    top: -4,
    right: -12,
    minWidth: 18,
    height: 18,
    borderRadius: radii.pill,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 4
  },
  badgeText: { fontSize: 10, lineHeight: 12 }
});
