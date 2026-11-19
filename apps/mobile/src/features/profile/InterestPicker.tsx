import React from "react";
import { StyleSheet, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import { spacing } from "../../theme/tokens";
import { Chip, LoadingState, Text } from "../../ui";

export const MAX_INTERESTS = 10;

export function InterestPicker({ selected, onChange }: { selected: number[]; onChange: (ids: number[]) => void }) {
  const { data, isLoading } = useQuery({ queryKey: queryKeys.interests, queryFn: profileApi.interests, staleTime: Infinity });
  if (isLoading || !data) return <LoadingState />;
  const toggle = (id: number) =>
    onChange(selected.includes(id) ? selected.filter((x) => x !== id) : selected.length < MAX_INTERESTS ? [...selected, id] : selected);
  return (
    <View style={styles.wrap}>
      <Text variant="caption" tone="subtle">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      <View style={styles.chips}>
        {data.map((i) => (
          <Chip
            key={i.id}
            label={i.label}
            selected={selected.includes(i.id)}
            disabled={!selected.includes(i.id) && selected.length >= MAX_INTERESTS}
            onPress={() => toggle(i.id)}
            testID={`interest-${i.slug}`}
          />
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
