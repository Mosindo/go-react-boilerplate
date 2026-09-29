import React from "react";
import { View } from "react-native";
import { useInterests } from "../../hooks/useProfileData";
import { messageFromError } from "../../lib/errors";
import { toggleInList, LIMITS } from "../../lib/validation";
import { ErrorView } from "../feedback/ErrorView";
import { Chip } from "../ui/Chip";
import { FormField } from "../ui/FormField";
import { Loader } from "../ui/Loader";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

type Props = {
  value: string[];
  onChange: (value: string[]) => void;
  error?: string | null;
};

const makeStyles = (t: Theme) => ({
  row: { flexDirection: "row" as const, flexWrap: "wrap" as const, gap: t.spacing.sm }
});

export function InterestPicker({ error, onChange, value }: Props) {
  const styles = useThemedStyles(makeStyles);
  const query = useInterests();

  return (
    <FormField
      error={error}
      hint={`${value.length}/${LIMITS.interestsMax}`}
      helperText={`Pick up to ${LIMITS.interestsMax}. They help us suggest people you'll click with.`}
      label="Interests"
    >
      {query.isPending ? <Loader label="Loading interests" size="small" /> : null}
      {query.isError ? (
        <ErrorView compact message={messageFromError(query.error)} onAction={() => void query.refetch()} />
      ) : null}
      {query.data ? (
        <View style={styles.row}>
          {query.data.map((interest) => {
            const selected = value.includes(interest.slug);
            return (
              <Chip
                disabled={!selected && value.length >= LIMITS.interestsMax}
                key={interest.slug}
                label={interest.label}
                onPress={() => onChange(toggleInList(value, interest.slug, LIMITS.interestsMax))}
                selected={selected}
                testID={`interest-${interest.slug}`}
              />
            );
          })}
        </View>
      ) : null}
    </FormField>
  );
}
