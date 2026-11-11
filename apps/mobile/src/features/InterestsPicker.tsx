import React, { useState } from "react";
import { View } from "react-native";
import type { Interest } from "../api/types";
import { useInterestCatalog } from "../hooks/queries";
import { Button, Chip, ErrorState, LoadingState, Text } from "../ui";

const MAX = 10;

type Props = {
  initial: Interest[];
  submitLabel: string;
  loading: boolean;
  error?: string | null;
  onSubmit: (slugs: string[]) => void;
};

export function InterestsPicker({ initial, submitLabel, loading, error, onSubmit }: Props) {
  const catalog = useInterestCatalog();
  const [selected, setSelected] = useState<string[]>(initial.map((i) => i.slug));

  if (catalog.isLoading) {
    return <LoadingState />;
  }
  if (catalog.isError) {
    return <ErrorState message="Impossible de charger les centres d'intérêt." onRetry={() => void catalog.refetch()} />;
  }

  const toggle = (slug: string) =>
    setSelected((prev) =>
      prev.includes(slug) ? prev.filter((s) => s !== slug) : prev.length >= MAX ? prev : [...prev, slug]
    );

  return (
    <View style={{ gap: 18 }}>
      <Text tone="muted">
        {selected.length}/{MAX} sélectionnés
      </Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
        {(catalog.data ?? []).map((i) => (
          <Chip
            key={i.slug}
            testID={`interest-${i.slug}`}
            label={i.label}
            selected={selected.includes(i.slug)}
            onPress={() => toggle(i.slug)}
          />
        ))}
      </View>
      {error ? <Text tone="danger">{error}</Text> : null}
      <Button testID="interests-submit" label={submitLabel} loading={loading} onPress={() => onSubmit(selected)} />
    </View>
  );
}
