#pragma once

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#include "../../geometry/matrix_types.h"
#include "../../geometry/rot3.h"
#include "../../geometry/vector_types.h"

#ifdef __cplusplus
extern "C" {
#endif

// TODO: split this up into several headers

typedef uint32_t BodyID;

typedef uint32_t SubShapeID;

// TODO: rename to PhysicsSystem and use "system" for param name
typedef struct Physics Physics;

typedef struct Shape Shape;

// TODO: rename
typedef struct MotionState MotionState;
struct MotionState {
	dvec3 position;
	Rot3 rotation;
	vec3 velocity;
	vec3 angularVelocity;
};

// TODO: rename to something like BodyProperties or BodyState or just Body or whatever...
typedef struct MotionProperties MotionProperties;
struct MotionProperties {
	const Shape *shape;

	MotionState motionState;

	int objectLayer;
	const BodyID *ignoreBodies;
	size_t ignoreBodyCount;

	int motionType;
	float gravityFactor;

	float mass;
	mat4 inertia;
};

typedef struct QueryFilter QueryFilter;
struct QueryFilter {
	BodyID ignore;
};

typedef enum ContactEventType {
	ContactAdded = 1,
	ContactRemoved,
} ContactEventType;

typedef struct ContactEvent ContactEvent;
struct ContactEvent {
	ContactEventType type;

	// TODO: name this struct?
	struct {
		BodyID bodyID;
		uint32_t subShapeID;
		bool active;
	} body1, body2;

	vec3 normal;
};

// TODO: better names for these things

Physics* newPhysics(
	int broadPhaseLayerCount,
	int objectLayerCount,
	const uint8_t *objectLayerToBroadPhaseLayer,
	const bool *shouldObjectLayersCollide);

// TODO: we'll have lots of different query type things be our physics API, and
// implement them in terms of whatever the physics engine can do.

typedef struct QueryShapeResult QueryShapeResult;
struct QueryShapeResult {
	dvec3 pos;
};

// casts also have a time to impact that we want in the result
typedef struct QueryHit QueryHit;
struct QueryHit {
	// TODO: just copy CollisionResult or whatever from jolt
	dvec3 point;
	vec3  normal;
	float depth;
};

typedef struct QueryResult QueryResult;
struct QueryResult {
	QueryHit *data;
	size_t len;
	size_t cap;
};

typedef struct CastQueryResult CastQueryResult;
struct CastQueryResult {
	float fraction;
	QueryHit base;
};

// TODO: pass base offset explicitly?

// TODO: change all queries to have an out parameter for query results

void physicsQueryCastRayClosestHit(
	Physics *system,
	const dvec3 *pos, const vec3 *dir);

size_t physicsQueryShape(
	Physics *system,
	const Shape *shape_, const dvec3 *pos, const Rot3 *rot, const vec3 *scale,
	const vec3 *movementDirection, float maxSeparationDistance,
	QueryFilter filter,
	QueryHit *outHits, size_t maxHitCount);

CastQueryResult physicsQuerySweptShapeClosestHit(
	Physics *physics,
	const Shape *shape, const dvec3 *pos, const Rot3 *rot, const vec3 *scale,
	const vec3 *displacement,
	QueryFilter filter);

void physicsSetGravity(Physics *physics, const vec3 *gravity);

void physicsAddBody(Physics *physics, BodyID bodyID, MotionProperties motionProperties);
void physicsUpdateBody(Physics *physics, BodyID bodyID, MotionProperties motionProperties);
void physicsRemoveBody(Physics *system, BodyID bodyID);
void physicsUpdate(Physics *physics, float deltaTime);
// TODO: merge this into physicsActiveBodies?
void physicsWritebackBody(Physics *physics, BodyID bodyID, MotionState *out);

// TODO: return structs?
const BodyID* physicsActiveBodies(Physics *system, size_t *activeBodyCount);
const ContactEvent* physicsContactEvents(Physics *physics, size_t *contactEventCount);

// TODO: a method for loading serialized (cached) shape

// TODO: replace our shape constructors with a single unified one, which will
// take a massive param struct?

// TODO: specify material

Shape* newSphereShape(float radius);
Shape* newBoxShape(const vec3 *halfExtent, float convexRadius);
Shape* newCylinderShape(float radius, float halfHeight, float convexRadius);
Shape* newConvexHullShape(const vec3 *points, size_t pointCount, float convexRadius);
// TODO: do it like physx -- pass materials deinterleaved from triangles?
Shape* newMeshShape(const vec3 *vertices, size_t vertexCount, const void *triangles, size_t triangleCount);
Shape* newTransformedShape(const vec3 *translation, const Rot3 *rotation, const vec3 *scale, const Shape *shape);
float shapeMass(const Shape *shape);
mat4 shapeInertia(const Shape *shape);
void shapeDecRef(Shape *shape);

#ifdef __cplusplus
}
#endif
