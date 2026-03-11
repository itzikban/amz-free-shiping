import { createAdminActionProxy } from "@/lib/api/adminProxyAction";

export const POST = createAdminActionProxy('/v1/admin/actions/retry-failed-notifications');
