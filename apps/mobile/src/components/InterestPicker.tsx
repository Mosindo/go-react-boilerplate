import React from "react";
import { StyleSheet, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { listInterests } from "../api/dating";
import { queryKeys } from "../lib/queryKeys";
import { Text } from "../shared/ui";
import { spacing } from "../shared/ui";
import { Chip } from "./Chip";

const MAX = 10;

type InterestPickerProps = {
  value: string[];
  onChange: (slugs: string[]) => void;
};

export function InterestPicker({ value, onChange }: InterestPickerProps) {
  const { data = [], isLoading, isError } = useQuery({ queryKey: queryKeys.interests, queryFn: listInterests, staleTime: Infinity });

  const toggle = (slug: string) => {
    if (value.includes(slug)) {
      onChange(value.filter((s) => s !== slug));
    } else if (value.length < MAX) {
      onChange([...value, slug]);
    }
  };

  if (isLoading) return <Text tone="muted">Chargement…</Text>;
  if (isError) return <Text tone="danger">Impossible de charger les centres d'intérêt.</Text>;

  return (
    <View>
      <View style={styles.wrap}>
        {data.map((interest) => (
          <Chip key={interest.slug} label={interest.label} onPress={() => toggle(interest.slug)} selected={value.includes(interest.slug)} />
        ))}
      </View>
      <Text style={styles.count} tone="muted" variant="caption">
        {value.length}/{MAX} sélectionnés
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  count: { marginTop: spacing.sm }
});
