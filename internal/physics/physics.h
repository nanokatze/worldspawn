#pragma once

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#include "gmath.h"

#ifdef __cplusplus
extern "C" {
#endif

// TODO: split this up into several headers?

typedef uint32_t BodyID;

typedef uint32_t SubShapeID;

// TODO: rename to PhysicsSystem or PhysicsScene
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

bool physicsFilterLayerImpl(void *pipeline, uint32_t layer);

bool physicsFilterBodyImpl(void *pipeline, BodyID bodyID);

// TODO: naming

typedef struct Ray Ray;
struct Ray {
	dvec3 origin;
	vec3 direction;
	float tmax;
};

typedef struct SceneRayHit SceneRayHit;
struct SceneRayHit {
	BodyID bodyID;
	float t;
	// SubShapeID subShapeID;
};

void physicsTraceRay(Physics *system, Ray ray, void *pipeline);

// TODO: we need a enum for this
int physicsRayHitImpl(void *pipeline, SceneRayHit hit);

// TODO: pass base offset explicitly?
typedef struct Overlap Overlap;
struct Overlap {
	dvec3 pos;
	Rot3 rot;
	vec3 scale;
	Shape *shape;
	// TODO: this is used by player controller atm but we should kill these
	vec3 movementDirection;
	float maxSeparationDistance;
};

typedef struct SceneOverlapHit SceneOverlapHit;
struct SceneOverlapHit {
	BodyID bodyID;
	vec3 contactPointOn1;
	vec3 contactPointOn2;
	vec3 penetrationAxis;
	float penetrationDepth;
	// subshape 1 and 2 here
};

// TODO: rename
void physicsOverlapQuery(Physics *system, Overlap overlap, void *pipeline);

int physicsOverlapHitTramp(void *pipeline, SceneOverlapHit hit);

typedef struct Sweep Sweep;
struct Sweep {
	dvec3 pos;
};

void physicsSweepQuery(Physics *system, Sweep sweep, void *pipeline);

void physicsSetGravity(Physics *physics, vec3 gravity);

void physicsAddBody(Physics *physics, BodyID bodyID, MotionProperties motionProperties);
void physicsUpdateBody(Physics *physics, BodyID bodyID, MotionProperties motionProperties);
void physicsRemoveBody(Physics *system, BodyID bodyID);
void physicsUpdate(Physics *physics, float deltaTime);
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
