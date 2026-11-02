import { sql } from 'drizzle-orm';
import { users, activityMinutes } from './db/schema';

export const hoursForUser = sql<number>`(select count(*)::float8 / 60 from ${activityMinutes} where ${activityMinutes.userId} = ${users.id})`;

// A streak remains current until the end of the next UTC day.
export const streakForUser = sql<number>`(select count(*)::int from (
  select day, max(day) over () as latest,
    (row_number() over (order by day desc))::int as position
  from (select distinct (to_timestamp(${activityMinutes.minute} * 60) at time zone 'UTC')::date as day
    from ${activityMinutes} where ${activityMinutes.userId} = ${users.id}) active_days
) days where latest >= (now() at time zone 'UTC')::date - 1
  and day = latest - (position - 1))`;

export const metrics = ['tokens', 'sessions', 'hours', 'streak'] as const;
export type Metric = typeof metrics[number];
export const parseMetric = (value: string | null): Metric => metrics.includes(value as Metric) ? value as Metric : 'tokens';
