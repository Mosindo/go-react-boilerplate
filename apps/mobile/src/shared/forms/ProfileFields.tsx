import React from "react";
import { View } from "react-native";
import { formatCounter } from "../../lib/format";
import { LIMITS, type ProfileFormErrors, type ProfileFormValues } from "../../lib/validation";
import { Input } from "../ui/Input";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";
import { GenderPicker } from "./GenderPicker";
import { InterestPicker } from "./InterestPicker";

type Props = {
  values: ProfileFormValues;
  onChange: (values: ProfileFormValues) => void;
  errors?: ProfileFormErrors;
  editable?: boolean;
};

const makeStyles = (t: Theme) => ({
  group: { gap: t.spacing.lg }
});

/** First name, gender and city. */
export function BasicsFields({ editable = true, errors, onChange, values }: Props) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View style={styles.group}>
      <Input
        autoCapitalize="words"
        autoComplete="given-name"
        editable={editable}
        error={errors?.firstName}
        label="First name"
        maxLength={LIMITS.firstNameMax}
        onChangeText={(firstName) => onChange({ ...values, firstName })}
        returnKeyType="next"
        testID="profile-firstname-input"
        textContentType="givenName"
        value={values.firstName}
      />
      <GenderPicker
        error={errors?.gender}
        label="I am"
        onChange={(gender) => onChange({ ...values, gender })}
        value={values.gender}
      />
      <Input
        autoCapitalize="words"
        editable={editable}
        error={errors?.city}
        helperText="Optional. Shown on your profile."
        label="City"
        maxLength={LIMITS.cityMax}
        onChangeText={(city) => onChange({ ...values, city })}
        testID="profile-city-input"
        textContentType="addressCity"
        value={values.city}
      />
    </View>
  );
}

/** Bio and interests. */
export function AboutFields({ editable = true, errors, onChange, values }: Props) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View style={styles.group}>
      <Input
        editable={editable}
        error={errors?.bio}
        hint={formatCounter(values.bio.length, LIMITS.bioMax)}
        label="About you"
        maxLength={LIMITS.bioMax}
        multiline
        onChangeText={(bio) => onChange({ ...values, bio })}
        placeholder="A few words about you, what you love, what you're looking for."
        testID="profile-bio-input"
        value={values.bio}
      />
      <InterestPicker
        error={errors?.interests}
        onChange={(interests) => onChange({ ...values, interests })}
        value={values.interests}
      />
    </View>
  );
}
