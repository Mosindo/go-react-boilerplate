import React from "react";
import { View } from "react-native";
import { strings } from "../../i18n/strings";
import { ChipGroup } from "../../shared/form";
import { ErrorView, LoadingView } from "../../shared/feedback";
import { Text } from "../../shared/ui";
import { useInterests } from "../../hooks/useProfile";
import { MAX_INTERESTS } from "../../utils/validation";
import { toggleInList } from "./options";

type InterestPickerProps = { selected: string[]; onChange: (slugs: string[]) => void };

export function InterestPicker({ onChange, selected }: InterestPickerProps) {
  const query = useInterests();

  if (query.isLoading) {
    return <LoadingView label={strings.common.loading} />;
  }
  if (query.isError || !query.data) {
    return <ErrorView message={strings.errors.generic} onAction={() => void query.refetch()} />;
  }

  return (
    <View style={{ gap: 10 }}>
      <ChipGroup
        disableUnselected={selected.length >= MAX_INTERESTS}
        onToggle={(slug) => onChange(toggleInList(selected, slug, MAX_INTERESTS))}
        options={query.data.map((interest) => ({ value: interest.slug, label: interest.label }))}
        selected={selected}
      />
      <Text tone="muted" variant="caption">
        {strings.profile.interestsCount(selected.length, MAX_INTERESTS)}
      </Text>
    </View>
  );
}
