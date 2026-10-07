import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getMyProfile,
  getPreferences,
  listBlocks,
  listInterests,
  savePreferences,
  saveMyProfile,
  unblockUser
} from "../api/profile";
import type { Preferences, ProfileInput, SelfProfile } from "../api/types";

export const profileKeys = {
  me: ["profile", "me"] as const,
  preferences: ["profile", "preferences"] as const,
  interests: ["profile", "interests"] as const,
  blocks: ["profile", "blocks"] as const
};

export function useMyProfile() {
  return useQuery<SelfProfile | null>({ queryKey: profileKeys.me, queryFn: getMyProfile });
}

export function usePreferences() {
  return useQuery<Preferences>({ queryKey: profileKeys.preferences, queryFn: getPreferences });
}

export function useInterests() {
  return useQuery({ queryKey: profileKeys.interests, queryFn: listInterests, staleTime: 60 * 60 * 1000 });
}

export function useBlockedUsers() {
  return useQuery({ queryKey: profileKeys.blocks, queryFn: listBlocks });
}

export function useSaveProfile() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: ProfileInput) => saveMyProfile(input),
    onSuccess: (profile) => client.setQueryData(profileKeys.me, profile)
  });
}

export function useSavePreferences() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: Preferences) => savePreferences(input),
    onSuccess: (preferences) => {
      client.setQueryData(profileKeys.preferences, preferences);
      // Discovery results depend on preferences: any cached feed is stale now.
      void client.invalidateQueries({ queryKey: ["discover"] });
    }
  });
}

export function useUnblock() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => unblockUser(userId),
    onSuccess: () => client.invalidateQueries({ queryKey: profileKeys.blocks })
  });
}
