import React from "react";
import { StyleSheet, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import { Chip, Loader, Text, spacing } from "../shared/ui";

export const MAX_INTERESTS = 10;

type InterestPickerProps = {
  selected: string[];
  onChange: (slugs: string[]) => void;
};

export function InterestPicker({ selected, onChange }: InterestPickerProps) {
  const { data, isLoading, isError } = useQuery({ queryKey: queryKeys.interests, queryFn: profileApi.interests, staleTime: Infinity });

  if (isLoading) {
    return <Loader />;
  }
  if (isError || !data) {
    return <Text tone="danger">Impossible de charger les centres d&apos;intérêt.</Text>;
  }

  const toggle = (slug: string) => {
    if (selected.includes(slug)) {
      onChange(selected.filter((s) => s !== slug));
    } else if (selected.length < MAX_INTERESTS) {
      onChange([...selected, slug]);
    }
  };

  return (
    <View style={styles.wrap}>
      {data.map((interest) => (
        <Chip
          disabled={!selected.includes(interest.slug) && selected.length >= MAX_INTERESTS}
          key={interest.slug}
          label={interest.label}
          onPress={() => toggle(interest.slug)}
          selected={selected.includes(interest.slug)}
        />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({ wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm } });
