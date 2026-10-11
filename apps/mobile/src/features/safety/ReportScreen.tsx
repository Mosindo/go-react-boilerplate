import { useState } from 'react';
import { View } from 'react-native';
import { useMutation } from '@tanstack/react-query';
import { REPORT_REASONS, safetyApi } from '../../api';
import type { ReportReason } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { REPORT_DETAILS_MAX } from '../../lib/validation';
import type { RootScreenProps } from '../../navigation/types';
import { useStyles } from '../../theme';
import { Button, Chip, Input, Screen, Text, useToast } from '../../ui';

export function ReportScreen({ navigation, route }: RootScreenProps<'Report'>) {
  const { userId, firstName } = route.params;
  const toast = useToast();
  const onError = useErrorToast();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState('');
  const [submitted, setSubmitted] = useState(false);
  const styles = useStyles((t) => ({
    chips: { gap: t.spacing.sm },
    group: { gap: t.spacing.sm },
  }));

  const report = useMutation({
    mutationFn: () =>
      safetyApi.report({
        userId,
        reason: reason as ReportReason,
        details: details.trim() ? details.trim() : undefined,
      }),
    onSuccess: () => {
      toast.success(fr.safety.reportSent);
      navigation.goBack();
    },
    onError,
  });

  const submit = () => {
    setSubmitted(true);
    if (reason) report.mutate();
  };

  return (
    <Screen scroll edges={['bottom']}>
      <Text variant="title">{fr.safety.reportTitle(firstName)}</Text>
      <Text color="textMuted">{fr.safety.reportIntro}</Text>
      <View style={styles.group} accessibilityRole="radiogroup">
        <Text variant="label">{fr.safety.reasonLabel}</Text>
        <View style={styles.chips}>
          {REPORT_REASONS.map((r) => (
            <Chip
              key={r}
              role="radio"
              label={fr.safety.reasons[r]}
              selected={reason === r}
              onPress={() => setReason(r)}
            />
          ))}
        </View>
        {submitted && !reason ? (
          <Text variant="caption" color="danger" accessibilityLiveRegion="polite">
            {fr.validation.reason}
          </Text>
        ) : null}
      </View>
      <Input
        label={fr.safety.details}
        helper={fr.safety.detailsHelper}
        value={details}
        onChangeText={setDetails}
        multiline
        maxLength={REPORT_DETAILS_MAX}
        showCounter
      />
      <Button label={fr.safety.sendReport} onPress={submit} loading={report.isPending} />
    </Screen>
  );
}
