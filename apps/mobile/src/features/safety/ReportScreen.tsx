import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { reportUser } from "../../api/social";
import { REPORT_REASONS, type ReportReason } from "../../api/types";
import type { MainStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Chip, Input, Screen, Text, useToast } from "../../ui";

type Props = NativeStackScreenProps<MainStackParamList, "Report">;

export function ReportScreen({ route, navigation }: Props) {
  const { userId, name } = route.params;
  const toast = useToast();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!reason) {
      setError("Choose a reason.");
      return;
    }
    setBusy(true);
    try {
      await reportUser(userId, reason, details.trim());
      toast.show("Thanks, your report was sent.");
      navigation.goBack();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not send the report.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll edges={["bottom"]}>
      <Text variant="heading">Report {name}</Text>
      <Text tone="muted" style={{ marginVertical: spacing.md }}>
        Reports are confidential. Our team reviews every one.
      </Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap" }}>
        {REPORT_REASONS.map((r) => (
          <Chip
            key={r.value}
            label={r.label}
            selected={reason === r.value}
            onPress={() => setReason(r.value)}
          />
        ))}
      </View>
      <Input
        label="Details (optional)"
        value={details}
        onChangeText={setDetails}
        multiline
        maxLength={1000}
        style={{ minHeight: 100, textAlignVertical: "top" }}
        error={error}
      />
      <Button label="Send report" onPress={() => void submit()} loading={busy} />
    </Screen>
  );
}
